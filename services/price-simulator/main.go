package main

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
)

type Security struct {
	Symbol string  `json:"symbol"`
	Price  float64 `json:"price"`
}

type PriceUpdate struct {
	Security string  `json:"security"`
	Price    float64 `json:"price"`
}

var (
	upgrader = websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool { return true }, // Allow all origins for this example
	}
	securityPrices = map[string]*Security{
		"AAPL": {Symbol: "AAPL", Price: 150.00},
		"GOOG": {Symbol: "GOOG", Price: 2500.00},
	}
	clientsMu sync.Mutex
	clients   = make(map[*websocket.Conn]bool)
)

func main() {
	r := mux.NewRouter()
	r.HandleFunc("/prices", handlePriceStream)
	go generatePrices()

	fmt.Println("Starting price stream server on :8080")
	if err := http.ListenAndServe(":8080", r); err != nil {
		panic(err)
	}
}

func handlePriceStream(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		fmt.Println("upgrade:", err)
		return
	}
	defer conn.Close()

	clientsMu.Lock()
	clients[conn] = true
	clientsMu.Unlock()

	defer func() {
		clientsMu.Lock()
		delete(clients, conn)
		clientsMu.Unlock()
	}()

	fmt.Println("Client connected")

	for {
		// Keep the connection alive (optional - for demonstration purposes)
		messageType, p, err := conn.ReadMessage()
		if err != nil {
			fmt.Println("read:", err)
			break // Client disconnected
		}

		if messageType == websocket.TextMessage {
			fmt.Printf("Received message: %s\n", p) //Handle any client messages if needed.
			// For example, you could enable client side filtering here.
		}

		if messageType == websocket.CloseMessage {
			fmt.Println("Client initiated close")
			break // Client disconnected
		}
	}
}

func generatePrices() {
	for {
		for _, security := range securityPrices {
			// Simulate price change (random walk)
			change := (rand.Float64() - 0.5) * security.Price * 0.01 // +/- 1% change
			security.Price += change

			// Ensure price stays positive
			if security.Price < 0 {
				security.Price = 0.01 // Small positive value
			}

			priceUpdate := PriceUpdate{
				Security: security.Symbol,
				Price:    security.Price,
			}

			// Broadcast price update to all connected clients.
			broadcastPrice(priceUpdate)
		}

		time.Sleep(1 * time.Second) // Update prices every second
	}
}

func broadcastPrice(update PriceUpdate) {
	message, _ := json.Marshal(update)
	clientsMu.Lock()
	for client := range clients {
		err := client.WriteMessage(websocket.TextMessage, message)
		if err != nil {
			fmt.Println("write:", err)
			// Client is likely disconnected, remove them.
			delete(clients, client)
		}
	}
	clientsMu.Unlock()
}

