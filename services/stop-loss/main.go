package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gorilla/mux"
)

const dbFileName = "/app/data/orders.db"

func main() {
	// --- Environment Variable Loading and Validation ---
	temporalAddress := os.Getenv("TEMPORAL_ADDRESS")
	if temporalAddress == "" {
		log.Fatal("TEMPORAL_ADDRESS environment variable is not set")
	}

	priceFeedWsURL := os.Getenv("PRICE_WS_URL")
	if priceFeedWsURL == "" {
		log.Fatal("PRICE_WS_URL environment variable is not set")
	}

	// --- Temporal Client ---
	temporalClient, err := WaitDialTemporal(temporalAddress, 10)
	if err != nil {
		log.Fatalf("Failed to connect to Temporal server at %s: %v", temporalAddress, err)
	}
	defer temporalClient.Close()
	log.Println("Connected to Temporal server")

	// --- Order Repo ---
	db, err := openSQLiteDB(dbFileName)
	if err != nil {
		log.Fatalf("Failed to initialize SQLite database: %v", err)
	}
	defer db.Close()

	err = createOrdersTable(db)
	if err != nil {
		log.Fatalf("Failed to create tables: %v", err)
	}
	log.Println("SQLite database initialized")

	orderRepo := NewOrdersRepoSQLite(db) // Use SQLite OrdersRepo
	log.Println("Order repository initialized (SQLite)")

	// --- Orders Workflow Service ---
	ordersWorkflowService := NewOrdersService(temporalClient, orderRepo)
	log.Println("Order service created")

	// --- Price Update Channel ---
	pricesChannel := make(chan PriceUpdate, 1024)
	log.Println("Price update channel created")

	// --- Start Price Ingestion Service ---
	priceIngestionService := NewPriceIngestionService(priceFeedWsURL, pricesChannel)
	priceIngestionService.Start()
	log.Println("Price ingestion service started")

	// --- Start Temporal Worker ---
	go StartLossOrderWorker(temporalClient, orderRepo)
	log.Println("Loss Order Temporal worker started")

	log.Println("Starting price change dispatcher")
	go StartPriceChangeDispatcher(temporalClient, orderRepo, pricesChannel)

	// --- Compile HTML Templates ---
	tpl, err := compileTemplates()
	if err != nil {
		log.Fatalf("Failed to compile templates: %v", err)
	}
	log.Println("Templates compiled successfully")

	// --- Web Server Setup ---
	webServer := NewWebServer(tpl, temporalClient, orderRepo, ordersWorkflowService)
	r := mux.NewRouter()
	webServer.SetupRoutes(r)

	port := "8080"
	serverAddr := fmt.Sprintf(":%s", port)
	log.Printf("Starting server on %s", serverAddr)

	server := &http.Server{
		Addr:         serverAddr,
		Handler:      r,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  30 * time.Second,
	}

	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("Server failed to start: %v", err)
	}

	log.Println("Server stopped.")
}
