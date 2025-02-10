package main

import (
	"context"
	"fmt"
	"log"

	"go.temporal.io/api/enums/v1"
	"go.temporal.io/sdk/client"
)

type ordersService struct {
	temporalClient client.Client
	repo           OrdersRepo
}

func NewOrdersService(client client.Client, repo OrdersRepo) OrderWorkflowService {
	return &ordersService{
		temporalClient: client,
		repo:           repo,
	}
}

func (os *ordersService) CreateOrder(ctx context.Context, order StopLossOrder) error {
	workflowOptions := client.StartWorkflowOptions{
		ID:                    fmt.Sprintf("stop-loss-workflow-%s", order.ID),
		TaskQueue:             "stop-loss-task-queue",
		WorkflowIDReusePolicy: enums.WORKFLOW_ID_REUSE_POLICY_ALLOW_DUPLICATE,
	}

	workflowRun, err := os.temporalClient.ExecuteWorkflow(ctx, workflowOptions, StopLossWorkflow, order)
	if err != nil {
		log.Printf("Failed to start StopLossWorkflow for order %s: %v", order.ID, err)
		return err
	}
	log.Printf("Started workflow for order ID: %s, WorkflowID: %s, RunID: %s", order.ID, workflowRun.GetID(), workflowRun.GetRunID())

	return nil
}

func (os *ordersService) CancelOrder(ctx context.Context, workflowID string) error {
	err := os.temporalClient.SignalWorkflow(ctx, workflowID, "", CancelOrderSignalName, nil)
	if err != nil {
		return err
	}

	return nil
}
