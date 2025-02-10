package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/worker"
	"go.temporal.io/sdk/workflow"
)

func StartLossOrderWorker(temporalClient client.Client, ordersRepo OrdersRepo) {
	w := worker.New(temporalClient, "stop-loss-task-queue", worker.Options{})
	w.RegisterWorkflow(StopLossWorkflow)
	w.RegisterActivity(ExecuteOrderActivity)

	// CreateOrderActivity as a closure, capturing orderRepo
	createOrderActivity := func(ctx context.Context, order StopLossOrder) error {
		return CreateOrderActivity(ctx, order, ordersRepo)
	}
	w.RegisterActivityWithOptions(createOrderActivity, activity.RegisterOptions{
		Name: "CreateOrderActivity",
	})

	// UpdateOrderStatusActivity as a closure, capturing orderRepo
	updateOrderStatusActivity := func(ctx context.Context, orderID string, status string) error {
		return UpdateOrderStatusActivity(ctx, orderID, status, ordersRepo)
	}
	w.RegisterActivityWithOptions(updateOrderStatusActivity, activity.RegisterOptions{
		Name: "UpdateOrderStatusActivity",
	})

	log.Println("Starting Temporal worker...")
	err := w.Run(worker.InterruptCh())
	if err != nil {
		log.Fatalf("Unable to start worker: %v", err)
	}
}

func StopLossWorkflow(ctx workflow.Context, order StopLossOrder) error {
	options := workflow.ActivityOptions{
		ScheduleToCloseTimeout: time.Minute * 5,
		HeartbeatTimeout:       time.Second * 30,
		RetryPolicy: &temporal.RetryPolicy{
			InitialInterval:    time.Second * 5,
			BackoffCoefficient: 2.0,
			MaximumInterval:    time.Minute * 1,
			MaximumAttempts:    5,
		},
	}
	ctx = workflow.WithActivityOptions(ctx, options)
	logger := workflow.GetLogger(ctx)
	logger.Info("StopLossWorkflow started", "orderID", order.ID, "security", order.Security, "stopPrice", order.StopPrice, "quantity", order.Quantity)

	workflowInfo := workflow.GetInfo(ctx)
	runID := workflowInfo.WorkflowExecution.RunID
	workflowID := workflowInfo.WorkflowExecution.ID

	// set this here prior to creation so the disatcher can signal to the worker
	order.WorkflowID = workflowID

	err := workflow.ExecuteActivity(ctx, CreateOrderActivity, order).Get(ctx, nil)
	if err != nil {
		logger.Error("Failed to create order", err)
		return fmt.Errorf("failed to create order: %v", err)
	}

	priceUpdateChannel := workflow.GetSignalChannel(ctx, PriceUpdateSignalName)
	cancelOrderChannel := workflow.GetSignalChannel(ctx, CancelOrderSignalName)

	// TODO: are these ok being just in memory values? seems dicey
	isOrderExecuted := order.Status == OrderStatusExecuted
	isOrderCancelled := order.Status == OrderStatusCancelled

	selector := workflow.NewSelector(ctx)

	// **Wrap the selector in a for loop to keep listening**
	for !isOrderExecuted && !isOrderCancelled {
		selector = workflow.NewSelector(ctx) // Re-create selector in each loop iteration

		selector.AddReceive(priceUpdateChannel, func(c workflow.ReceiveChannel, more bool) {
			var signalData PriceUpdateSignalData
			c.Receive(ctx, &signalData)

			if signalData.Security != order.Security {
				logger.Warn("Received price update for incorrect security", "expected", order.Security, "received", signalData.Security)
				return
			}

			currentPrice := signalData.Price
			logger.Info("Received price update", "security", signalData.Security, "price", currentPrice, "stopPrice", order.StopPrice, "isOrderExecuted", isOrderExecuted, "isOrderCancelled", isOrderCancelled)

			if currentPrice <= order.StopPrice && !isOrderExecuted && !isOrderCancelled {
				logger.Info("Stop-loss price reached 📉!", "security", order.Security, "currentPrice", currentPrice, "stopPrice", order.StopPrice)
				isOrderExecuted = true

				var executionResult string
				err := workflow.ExecuteActivity(ctx, ExecuteOrderActivity, order.Security, order.Quantity).Get(ctx, &executionResult)
				if err != nil {
					logger.Error("ExecuteOrderActivity failed", "error", err)
					workflow.ExecuteActivity(ctx, UpdateOrderStatusActivity, order.ID, OrderStatusPending)
					return
				}

				logger.Info("ExecuteOrderActivity completed", "result", executionResult)

				err = workflow.ExecuteActivity(ctx, UpdateOrderStatusActivity, order.ID, OrderStatusExecuted).Get(ctx, nil)
				if err != nil {
					logger.Error("Failed to update order status to EXECUTED after execution", "error", err)
					return // Log error but execution is already done.
				}

				logger.Info("StopLossWorkflow executed for order", "orderID", order.ID, "runID", runID)
				// **Do NOT return here - exit loop instead**
				// return // Exit workflow after execution - REMOVED
			} else if currentPrice > order.StopPrice {
				logger.Debug("Price above stop-loss, waiting for trigger", "security", order.Security, "currentPrice", currentPrice, "stopPrice", order.StopPrice)
			}
		})

		selector.AddReceive(cancelOrderChannel, func(c workflow.ReceiveChannel, more bool) {
			logger.Info("Cancellation signal received for order", "orderID", order.ID, "runID", runID)
			isOrderCancelled = true
			err := workflow.ExecuteActivity(ctx, UpdateOrderStatusActivity, order.ID, OrderStatusCancelled).Get(ctx, nil)
			if err != nil {
				logger.Error("Failed to update order status to CANCELLED", "error", err)
			}
			logger.Info("StopLossWorkflow cancelled for order", "orderID", order.ID, "runID", runID)
			// **Do NOT return here - exit loop instead**
			// return // Exit workflow on cancellation - REMOVED
		})

		// Wait for either price signal OR cancellation signal within the selector:
		selector.Select(ctx)
		logger.Debug("Selector completed, looping again or exiting...") // Debug log to track loop iterations
	}

	logger.Info("StopLossWorkflow exiting selector loop and workflow", "orderID", order.ID, "runID", runID, "isOrderExecuted", isOrderExecuted, "isOrderCancelled", isOrderCancelled)
	return nil // Workflow completes after loop exits (execution or cancellation)
}

// ExecuteOrderActivity - remains the same
func ExecuteOrderActivity(ctx context.Context, security string, quantity int) (string, error) {
	log.Printf("Executing order for %d shares of %s", quantity, security)
	time.Sleep(2 * time.Second) // Simulate order execution delay - this is a mock implementation
	executionResult := fmt.Sprintf("Order for %d shares of %s executed successfully", quantity, security)
	return executionResult, nil
}

func CreateOrderActivity(ctx context.Context, order StopLossOrder, ordersRepo OrdersRepo) error {
	log.Printf("Creating order: ", order)
	_, err := ordersRepo.CreateOrder(order)
	if err != nil {
		return fmt.Errorf("failed to create order %s", order.ID, err)
	}
	return nil
}

func UpdateOrderStatusActivity(ctx context.Context, orderID string, status string, ordersRepo OrdersRepo) error {
	log.Printf("Updating order %s status to: %s", orderID, status)
	err := ordersRepo.UpdateOrderStatus(orderID, status)
	if err != nil {
		return fmt.Errorf("failed to update order status for order %s to %s: %w", orderID, status, err)
	}
	return nil
}

