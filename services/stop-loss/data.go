package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"

	_ "modernc.org/sqlite"
)

// SQLiteOrderStore struct to interact with SQLite database
type SQLiteOrderStore struct {
	db *sql.DB
}

// NewSQLiteOrderStore constructor
func NewSQLiteOrderStore(db *sql.DB) (*SQLiteOrderStore, error) {
	return &SQLiteOrderStore{db: db}, nil
}

func (s *SQLiteOrderStore) Create(order *Order) error {
	res, err := s.db.Exec(`
		INSERT INTO orders (security, stop_loss_price, quantity, status, placed_at)
		VALUES (?, ?, ?, ?, ?)
	`, order.Security, order.StopLossPrice, order.Quantity, order.Status, order.PlacedAt)
	if err != nil {
		return fmt.Errorf("failed to insert order: %w", err)
	}

	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected after insert: %w", err)
	}
	if rowsAffected != 1 {
		return fmt.Errorf("expected 1 row affected, got %d", rowsAffected) // More robust check
	}
	return nil
}

func (s *SQLiteOrderStore) Get() ([]Order, error) {
	rows, err := s.db.Query(`
		SELECT id, security, stop_loss_price, quantity, status, placed_at
		FROM orders
		ORDER BY placed_at DESC
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to query orders: %w", err)
	}
	defer rows.Close()

	var orders []Order
	for rows.Next() {
		var order Order
		if err := rows.Scan(&order.ID, &order.Security, &order.StopLossPrice, &order.Quantity, &order.Status, &order.PlacedAt); err != nil {
			return nil, fmt.Errorf("failed to scan order row: %w", err)
		}
		orders = append(orders, order)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error during rows iteration: %w", err)
	}
	return orders, nil
}

func (s *SQLiteOrderStore) Cancel(id int) error {
	res, err := s.db.Exec(`
		UPDATE orders SET status = 'canceled' WHERE id = ?
	`, id)
	if err != nil {
		return fmt.Errorf("failed to cancel order: %w", err)
	}
	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected after cancel: %w", err)
	}
	if rowsAffected != 1 {
		return fmt.Errorf("expected 1 row affected when canceling order id %d, got %d", id, rowsAffected) // More robust check
	}

	return nil
}

func (s *SQLiteOrderStore) UpdateStatus(status string) error {
	// For now, not implementing generic status updates via this method
	// Status updates will be more specific (e.g., 'triggered' in a separate worker)
	return fmt.Errorf("UpdateStatus method not implemented for SQLiteOrderStore")
}

func initDatabase(dbPath string) (*sql.DB, error) {
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Check if the database file exists. If not, migrations should create it.
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		log.Println("Database file does not exist. Assuming migrations will create it.")
	} else if err != nil {
		return nil, fmt.Errorf("failed to stat database file: %w", err)
	} else {
		log.Println("Database file exists.")
	}

	if err := db.Ping(); err != nil {
		db.Close() // Close if ping fails
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	log.Println("Successfully connected to database.")
	return db, nil
}

func NewOrderStore(db *sql.DB) (OrderStore, error) {
	return NewSQLiteOrderStore(db)
}
