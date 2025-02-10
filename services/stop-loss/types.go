package main

import "time"

// StopLossOrder struct - aligned with web.go and worker code
type StopLossOrder struct {
	ID         string    `json:"id"`
	Security   string    `json:"security"`
	StopPrice  float64   `json:"stopPrice"` // Changed to StopPrice and float64
	Quantity   int       `json:"quantity"`
	Status     string    `json:"status"`               // pending, executed, cancelled - using constants below
	PlacedAt   time.Time `json:"placedAt"`             // Aligned field name
	WorkflowID string    `json:"workflowID,omitempty"` // Temporal Workflow ID
}

// OrderServiceInterface - renamed and aligned with implementation and usage
type OrderServiceInterface interface {
	CreateOrder(order StopLossOrder) (StopLossOrder, error)
	GetOrder(orderID string) (StopLossOrder, error)
	CancelOrder(orderID string) error
	ListOrders() ([]StopLossOrder, error)
	UpdateOrderStatus(orderID string, status string) error
	AssociateWorkflowID(orderID string, workflowID string) error
	GetWorkflowIDsForSecurity(security string) []string
	GetOrdersForSecurity(security string) []StopLossOrder
}

// PriceUpdateSignal is the signal type for price updates.
type PriceUpdateSignal struct {
	Price float64
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
