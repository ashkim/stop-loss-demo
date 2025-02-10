package main

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"os"
	"strconv"
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
		CheckOrigin: func(r *http.Request) bool { return true }, // Allow all origins
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

	disruptionProbabilityStr := os.Getenv("DISRUPTION_PROBABILITY")
	disruptionProbability := 0.0 // Default: no disruption
	if prob, err := strconv.ParseFloat(disruptionProbabilityStr, 64); err == nil {
		disruptionProbability = prob
	}
	fmt.Printf("Price simulator started with disruption probability: %.2f\n", disruptionProbability)

	go generatePrices()                          // Goroutine for price generation and broadcasting
	go disruptConnections(disruptionProbability) // Separate goroutine for disruption simulation

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
		messageType, _, err := conn.ReadMessage() // Discard read messages
		if err != nil {
			fmt.Println("read error:", err)
			break // Exit loop on read error (client disconnect)
		}

		if messageType == websocket.CloseMessage {
			fmt.Println("Client initiated close")
			break // Client disconnected
		}
	}
}

func generatePrices() {
	fmt.Println("Starting price generator...")
	for {
		for _, security := range securityPrices {
			// Simulate price change (random walk)
			change := (rand.Float64() - 0.5) * security.Price * 0.01 // +/- 1% change
			security.Price += change

			// Keep price positive
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

// disruptConnections simulates random client disconnections based on probability.
func disruptConnections(disruptionProbability float64) {
	fmt.Println("Starting connection disruptor...")
	if disruptionProbability <= 0 {
		fmt.Println("Disruptions disabled (probability <= 0)")
		return // Exit if disruption probability is not positive
	}

	for {
		// Simulate disruption probabilistically
		if rand.Float64() < disruptionProbability {
			simulateDisruption()
		}
		time.Sleep(5 * time.Second) // Check for disruptions less frequently than price updates (e.g., every 5 seconds)
	}
}

func simulateDisruption() {
	clientsMu.Lock()
	defer clientsMu.Unlock()

	if len(clients) > 0 {
		// Select a random client to disconnect
		var clientToDisconnect *websocket.Conn = nil
		clientIndex := rand.Intn(len(clients))
		index := 0
		for client := range clients {
			if index == clientIndex {
				clientToDisconnect = client
				break
			}
			index++
		}

		if clientToDisconnect != nil {
			fmt.Println("Simulating disruption - closing connection to a client.")
			clientToDisconnect.Close()          // Simulate abrupt closure
			delete(clients, clientToDisconnect) // Remove client from active list
		}
	}
}

func broadcastPrice(update PriceUpdate) {
	message, _ := json.Marshal(update)
	clientsMu.Lock()
	for client := range clients {
		err := client.WriteMessage(websocket.TextMessage, message)
		if err != nil {
			fmt.Println("write error:", err)
			// Client likely disconnected or in bad state, remove them.
			delete(clients, client)
		}
	}
	clientsMu.Unlock()
}

