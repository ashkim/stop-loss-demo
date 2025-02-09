package main

import (
	"fmt"
	"log"
	"net/http"
	"time"
)

func main() {
	if err := compileTemplates(); err != nil {
		log.Fatalf("Failed to compile templates: %v", err)
	}
	log.Println("Templates compiled successfully")

	mux := http.NewServeMux()
	SetupRoutes(mux)

	port := "8080"
	serverAddr := fmt.Sprintf(":%s", port)
	log.Printf("Starting server on %s", serverAddr)

	server := &http.Server{
		Addr:         serverAddr,
		Handler:      mux,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  30 * time.Second,
	}

	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("Server failed to start: %v", err)
	}

	log.Println("Server stopped.")

	select {} // Keep main goroutine alive
}
