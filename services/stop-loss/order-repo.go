package main

import (
	"database/sql"
	"errors"
	"fmt"
	"log"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

type OrdersRepoSQLite struct {
	db *sql.DB
}

func NewOrdersRepoSQLite(db *sql.DB) *OrdersRepoSQLite {
	return &OrdersRepoSQLite{
		db: db,
	}
}

func (s *OrdersRepoSQLite) CreateOrder(order StopLossOrder) (StopLossOrder, error) {
	_, err := s.db.Exec(`
		INSERT INTO orders (id, security, stop_price, quantity, status, placed_at, workflow_id)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`, order.ID, order.Security, order.StopPrice, order.Quantity, order.Status, order.PlacedAt, order.WorkflowID)
	if err != nil {
		return StopLossOrder{}, fmt.Errorf("failed to create order in database: %w", err)
	}
	return order, nil
}

func (s *OrdersRepoSQLite) GetOrder(orderID string) (StopLossOrder, error) {
	row := s.db.QueryRow(`SELECT id, security, stop_price, quantity, status, placed_at, workflow_id FROM orders WHERE id = ?`, orderID)
	var order StopLossOrder
	var placedAt string // SQLite stores DATETIME as TEXT
	err := row.Scan(&order.ID, &order.Security, &order.StopPrice, &order.Quantity, &order.Status, &placedAt, &order.WorkflowID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return StopLossOrder{}, errors.New("order not found")
		}
		return StopLossOrder{}, fmt.Errorf("failed to get order from database: %w", err)
	}

	// Parse PlacedAt from string to time.Time
	parseTime, err := time.Parse(time.RFC3339, placedAt)
	if err != nil {
		log.Printf("Error parsing placed_at from database: %v", err)
		return StopLossOrder{}, fmt.Errorf("error parsing placed_at: %w", err) // Or handle more gracefully if needed
	}
	order.PlacedAt = parseTime

	return order, nil
}

func (s *OrdersRepoSQLite) CancelOrder(orderID string) error {
	res, err := s.db.Exec(`UPDATE orders SET status = ? WHERE id = ? AND status = ?`, OrderStatusCancelled, orderID, OrderStatusPending)
	if err != nil {
		return fmt.Errorf("failed to cancel order in database: %w", err)
	}
	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get affected rows on cancel: %w", err)
	}
	if rowsAffected == 0 {
		// Check if order exists, if not return "not found", otherwise "not pending"
		_, err := s.GetOrder(orderID)
		if err != nil {
			return errors.New("order not found") // Or original error if you want to be more specific
		}
		return errors.New("order is not pending and cannot be cancelled") // Order exists but not pending
	}
	return nil
}

func (s *OrdersRepoSQLite) ListOrders() ([]StopLossOrder, error) {
	rows, err := s.db.Query(`SELECT id, security, stop_price, quantity, status, placed_at, workflow_id FROM orders`)
	if err != nil {
		return nil, fmt.Errorf("failed to list orders from database: %w", err)
	}
	defer rows.Close()

	var orders []StopLossOrder
	for rows.Next() {
		var order StopLossOrder
		var placedAt string
		err := rows.Scan(&order.ID, &order.Security, &order.StopPrice, &order.Quantity, &order.Status, &placedAt, &order.WorkflowID)
		if err != nil {
			return nil, fmt.Errorf("error scanning order row: %w", err)
		}
		// Parse PlacedAt from string to time.Time
		parseTime, err := time.Parse(time.RFC3339, placedAt)
		if err != nil {
			log.Printf("Error parsing placed_at from database: %v", err) // Log and continue, or return error?
			continue                                                     // Let's continue and log, for now
		}
		order.PlacedAt = parseTime
		orders = append(orders, order)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating order rows: %w", err)
	}
	return orders, nil
}

func (s *OrdersRepoSQLite) UpdateOrderStatus(orderID string, status string) error {
	_, err := s.db.Exec(`UPDATE orders SET status = ? WHERE id = ?`, status, orderID)
	if err != nil {
		return fmt.Errorf("failed to update order status in database: %w", err)
	}
	return nil
}

func (s *OrdersRepoSQLite) AssociateWorkflowID(orderID string, workflowID string) error {
	_, err := s.db.Exec(`UPDATE orders SET workflow_id = ? WHERE id = ?`, workflowID, orderID)
	if err != nil {
		return fmt.Errorf("failed to associate workflow ID in database: %w", err)
	}
	return nil
}

func (s *OrdersRepoSQLite) GetPendingWorkflowIDsForSecurity(security string) ([]string, error) {
	rows, err := s.db.Query(`SELECT workflow_id FROM orders WHERE security = ? AND workflow_id IS NOT NULL AND status is ?`, security, OrderStatusPending)
	if err != nil {
		return nil, fmt.Errorf("failed to get workflow IDs for security from database: %w", err)
	}
	defer rows.Close()

	var workflowIDs []string
	for rows.Next() {
		var workflowID string
		err := rows.Scan(&workflowID)
		if err != nil {
			log.Printf("Error scanning workflow ID row: %v", err)
			continue // Log and continue, or return error?
		}
		workflowIDs = append(workflowIDs, workflowID)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating workflow ID rows: %w", err)
	}
	return workflowIDs, nil
}

func (s *OrdersRepoSQLite) GetOrdersForSecurity(security string) ([]StopLossOrder, error) { // Added error return
	rows, err := s.db.Query(`SELECT id, security, stop_price, quantity, status, placed_at, workflow_id FROM orders WHERE security = ?`, security)
	if err != nil {
		return nil, fmt.Errorf("failed to get orders for security from database: %w", err)
	}
	defer rows.Close()

	var orders []StopLossOrder
	for rows.Next() {
		var order StopLossOrder
		var placedAt string
		err := rows.Scan(&order.ID, &order.Security, &order.StopPrice, &order.Quantity, &order.Status, &placedAt, &order.WorkflowID)
		if err != nil {
			return nil, fmt.Errorf("error scanning order row: %w", err)
		}
		// Parse PlacedAt from string to time.Time
		parseTime, err := time.Parse(time.RFC3339, placedAt)
		if err != nil {
			log.Printf("Error parsing placed_at from database: %v", err)
			continue // Log and continue
		}
		order.PlacedAt = parseTime
		orders = append(orders, order)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating order rows: %w", err)
	}
	return orders, nil
}

// Ensure OrdersRepoSQLite implements OrdersRepo
var _ OrdersRepo = (*OrdersRepoSQLite)(nil)
