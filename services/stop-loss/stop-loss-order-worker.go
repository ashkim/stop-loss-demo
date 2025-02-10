package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/worker"
	"go.temporal.io/sdk/workflow"
)

func StopLossWorkflow(ctx workflow.Context, orderID string, security string, stopPrice float64, quantity int) (string, error) {
	fmt.Println("StopLossWorkflow started for order:", orderID)
	// Placeholder - This is where your stop-loss logic will go
	fmt.Println("StopLossWorkflow complete for order:", orderID)
	return "Stop-loss workflow completed", nil
}

func ExecuteOrderActivity(ctx context.Context, security string, quantity int) (string, error) {
	fmt.Println("Executing order for", quantity, "units of", security)
	// Placeholder - This is where you'd integrate with an exchange API (mocked for now)
	return "Order executed successfully", nil
}

func WaitDialTemporal(hostAddress string, connectionRetryAttempts int) (client.Client, error) {
	var c client.Client
	var err error

	for i := 0; i < connectionRetryAttempts; i++ {
		c, err = client.Dial(client.Options{
			HostPort: hostAddress,
		})
		if err == nil {
			// success
			return c, nil
		}
		log.Printf("Failed to connect to Temporal (attempt %d): %v", i+1, err)
		time.Sleep(5 * time.Second)
	}

	return nil, err
}

func StartWorker(c client.Client) {
	w := worker.New(c, "stop-loss-task-queue", worker.Options{}) // Use a descriptive task queue name
	w.RegisterWorkflow(StopLossWorkflow)
	w.RegisterActivity(ExecuteOrderActivity)
	go w.Run(worker.InterruptCh())

	// You can start a workflow from here for testing, or you can have another service that starts workflows.
	// Example starting a workflow (remove this for production, start workflows through API):
	workflowOptions := client.StartWorkflowOptions{
		ID:        "stop-loss-workflow-id-1", // Unique workflow ID
		TaskQueue: "stop-loss-task-queue",    // Same task queue as the worker
	}
	_, err := c.ExecuteWorkflow(context.Background(), workflowOptions, StopLossWorkflow, "order-123", "AAPL", 150.00, 100)
	if err != nil {
		log.Fatalln("Unable to execute workflow:", err)
	}

	select {} // Keep the worker running
}
