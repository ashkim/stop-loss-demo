package main

type Order struct {
	ID       int     `json:"id"`
	Security string  `json:"security"`
	Price    float64 `json:"price"`
	Quantity int     `json:"quantity"`
	Status   string  `json:"status"` // pending, executed, cancelled
}

// DataStore interface
type DataStore interface {
	CreateOrder(order *Order) error
	GetOrders() ([]Order, error)
	// ... other data access methods
}
