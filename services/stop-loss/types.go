package main

type Order struct {
	ID       int     `json:"id"`
	Security string  `json:"security"`
	Price    float64 `json:"price"`
	Quantity int     `json:"quantity"`
	Status   string  `json:"status"` // pending, executed, cancelled
}

type DataStore interface {
	CreateOrder(order *Order) error
	GetOrders() ([]Order, error)
	CancelOrder(id int) error
	UpdateOrderStatus(status string) error
}
