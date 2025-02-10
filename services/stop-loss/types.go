package main

import (
	"context"
	"time"

	"github.com/gorilla/websocket"
)

type StopLossOrder struct {
	ID         string    `json:"id"`
	Security   string    `json:"security"`
	StopPrice  float64   `json:"stopPrice"`
	Quantity   int       `json:"quantity"`
	Status     string    `json:"status"` // pending, executed, cancelled - using constants below
	PlacedAt   time.Time `json:"placedAt"`
	WorkflowID string    `json:"workflowID,omitempty"` // Temporal Workflow ID
}

type OrderWorkflowService interface {
	CreateOrder(ctx context.Context, order StopLossOrder) error
	CancelOrder(ctx context.Context, orderID string) error
}

type OrdersRepo interface {
	CreateOrder(order StopLossOrder) (StopLossOrder, error)
	GetOrder(orderID string) (StopLossOrder, error)
	CancelOrder(orderID string) error
	ListOrders() ([]StopLossOrder, error)
	UpdateOrderStatus(orderID string, status string) error
	AssociateWorkflowID(orderID string, workflowID string) error
	GetPendingWorkflowIDsForSecurity(security string) ([]string, error)
	GetOrdersForSecurity(security string) ([]StopLossOrder, error)
}

// PriceIngestionService manages the WebSocket connection and price updates.
type PriceIngestionService struct {
	wsURL         string
	conn          *websocket.Conn
	pricesChannel chan PriceUpdate // Channel to publish price updates
}

// PriceUpdate struct to hold price update information
type PriceUpdate struct {
	Security string  `json:"security"`
	Price    float64 `json:"price"`
}

// PriceUpdateSignal is the signal type for price updates.
type PriceUpdateSignal struct {
	Data PriceUpdateSignalData
}

type PriceUpdateSignalData struct {
	Security string  `json:"security"`
	Price    float64 `json:"price"`
}

// CancelOrderSignal is the signal type for order cancellation.
type CancelOrderSignal struct{}

// WorkflowSignals to keep signal names as constants
const (
	PriceUpdateSignalName = "priceUpdate"
	CancelOrderSignalName = "cancelOrder"
)

// Workflow statuses
const (
	OrderStatusPending   = "PENDING"
	OrderStatusExecuted  = "EXECUTED"
	OrderStatusCancelled = "CANCELLED"
)
