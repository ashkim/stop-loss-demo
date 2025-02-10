package main

import (
	"encoding/json"
	"log"
	"time"

	"github.com/gorilla/websocket"
)

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

// NewPriceIngestionService creates a new PriceIngestionService.
func NewPriceIngestionService(wsURL string, pricesChannel chan PriceUpdate) *PriceIngestionService {
	return &PriceIngestionService{
		wsURL:         wsURL,
		pricesChannel: pricesChannel,
	}
}

// Start starts the PriceIngestionService, establishing WebSocket connection and handling reconnection.
func (pis *PriceIngestionService) Start() {
	log.Println("Starting Price Ingestion Service...")
	go pis.run()
}

func (pis *PriceIngestionService) run() {
	var reconnectInterval = time.Second // Initial reconnect interval

	for {
		log.Println("Attempting to connect to WebSocket...")
		conn, _, err := websocket.DefaultDialer.Dial(pis.wsURL, nil)
		if err != nil {
			log.Printf("Failed to connect to WebSocket: %v. Retrying in %s...", err, reconnectInterval)
			time.Sleep(reconnectInterval)
			reconnectInterval = minDuration(reconnectInterval*2, 10*time.Second) // Exponential backoff, max 10s
			continue
		}

		log.Println("WebSocket connected.")
		pis.conn = conn
		reconnectInterval = time.Second // Reset reconnect interval on successful connection

		log.Println("Successfully subscribed to securities after reconnection.")
		pis.startReceivingPrices()
	}
}

func (pis *PriceIngestionService) startReceivingPrices() {
	defer pis.closeConnection()

	for {
		_, message, err := pis.conn.ReadMessage()
		if err != nil {
			log.Println("WebSocket ReadMessage error:", err) // More general read error log
			if websocket.IsCloseError(err, websocket.CloseNormalClosure, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Println("WebSocket connection closed by remote side (expected close error).")
			} else {
				log.Printf("WebSocket connection error (not close error): %v. Reconnecting...", err) // Specific log for non-close errors
			}
			return // Exit to trigger reconnection attempt in run()
		}

		var priceUpdate PriceUpdate
		if err := json.Unmarshal(message, &priceUpdate); err != nil {
			log.Printf("Error unmarshalling price update: %v, message: %s", err, string(message))
			continue
		}

		//log.Printf("Received price update: Security=%s, Price=%.2f", priceUpdate.Security, priceUpdate.Price)
		pis.pricesChannel <- priceUpdate
	}
}

func (pis *PriceIngestionService) closeConnection() {
	if pis.conn != nil {
		log.Println("Closing WebSocket connection.")
		if err := pis.conn.Close(); err != nil {
			log.Printf("Error closing WebSocket connection: %v", err)
		}
		pis.conn = nil
	}
}

func minDuration(d1, d2 time.Duration) time.Duration {
	if d1 < d2 {
		return d1
	}
	return d2
}
