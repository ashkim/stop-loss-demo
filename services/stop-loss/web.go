package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/mux"
	"go.temporal.io/sdk/client"
)

// WebServer struct now holds template, temporal client, and order service
type WebServer struct {
	template       *template.Template
	temporalClient client.Client
	orderService   OrderServiceInterface // Use the interface
}

// NewWebServer constructor
func NewWebServer(tpl *template.Template, tc client.Client, os OrderServiceInterface) *WebServer {
	return &WebServer{
		template:       tpl,
		temporalClient: tc,
		orderService:   os,
	}
}

// SetupRoutes configures the HTTP routes using methods on WebServer
func (s *WebServer) SetupRoutes(mux *mux.Router) {
	mux.HandleFunc("/", s.handleIndex).Methods("GET")
	mux.HandleFunc("/orders", s.handleCreateOrder).Methods("POST")
	mux.HandleFunc("/orders", s.handleListOrders).Methods("GET")               // redundant? consider removing
	mux.HandleFunc("/orders/{id}/cancel", s.handleCancelOrder).Methods("POST") // Cancellation endpoint
	mux.HandleFunc("/orders/{id}", s.handleGetOrder).Methods("GET")
	mux.HandleFunc("/sse-orders", s.handleSSEOrders).Methods("GET") // SSE endpoint
}

// handleIndex renders the main index page
func (s *WebServer) handleIndex(w http.ResponseWriter, r *http.Request) {
	log.Println("Loading index page")
	orders, err := s.orderService.ListOrders()
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to load orders: %v", err), http.StatusInternalServerError)
		return
	}
	data := IndexPageData{Orders: orders}
	err = s.template.ExecuteTemplate(w, "layout.html", data)
	if err != nil {
		http.Error(w, fmt.Sprintf("Template execution error: %v", err), http.StatusInternalServerError)
	}
}

// handleCreateOrder handles order placement via form submission
func (s *WebServer) handleCreateOrder(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Error parsing form", http.StatusBadRequest)
		return
	}

	security := r.FormValue("security")
	priceStr := r.FormValue("price")
	quantityStr := r.FormValue("quantity")

	price, err := strconv.ParseFloat(priceStr, 64)
	if err != nil {
		http.Error(w, "Invalid price", http.StatusBadRequest)
		return
	}
	quantity, err := strconv.Atoi(quantityStr)
	if err != nil {
		http.Error(w, "Invalid quantity", http.StatusBadRequest)
		return
	}

	orderID := fmt.Sprintf("order-%d", time.Now().UnixNano()) // Simple unique ID
	order := StopLossOrder{
		ID:        orderID,
		Security:  security,
		StopPrice: price,
		Quantity:  quantity,
		Status:    OrderStatusPending, // Initial status
	}

	createdOrder, err := s.orderService.CreateOrder(order)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to create order: %v", err), http.StatusInternalServerError)
		return
	}

	workflowOptions := client.StartWorkflowOptions{
		ID:                    fmt.Sprintf("stop-loss-workflow-%s", orderID), // Unique workflow ID
		TaskQueue:             "stop-loss-task-queue",                        // Same task queue as worker
		WorkflowIDReusePolicy: client.WorkflowIDReusePolicyAllowDuplicate,
	}

	workflowRun, err := s.temporalClient.ExecuteWorkflow(r.Context(), workflowOptions, StopLossWorkflow, createdOrder.ID, createdOrder.Security, createdOrder.StopPrice, createdOrder.Quantity)
	if err != nil {
		// On workflow start failure, consider reverting order creation or marking it as failed in service
		log.Printf("Failed to start StopLossWorkflow for order %s: %v", createdOrder.ID, err)
		http.Error(w, "Failed to start order workflow", http.StatusInternalServerError)
		return
	}
	log.Printf("Started workflow for order ID: %s, WorkflowID: %s, RunID: %s", createdOrder.ID, workflowRun.GetID(), workflowRun.GetRunID())

	err = s.orderService.AssociateWorkflowID(createdOrder.ID, workflowRun.GetID())
	if err != nil {
		log.Printf("Failed to associate workflow ID with order %s: %v", createdOrder.ID, err)
		// Log the error, but workflow is already running. Consider implications for tracking/cancellation if association fails.
	}

	// Respond with updated order list via SSE - or just redirect to refresh the order list
	s.renderOrderList(w, r) // Re-render and send updated order list

}

// handleListOrders fetches and renders the list of orders (used by SSE and initial load)
func (s *WebServer) handleListOrders(w http.ResponseWriter, r *http.Request) {
	s.renderOrderList(w, r)
}

func (s *WebServer) renderOrderList(w http.ResponseWriter, r *http.Request) {
	orders, err := s.orderService.ListOrders()
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to load orders: %v", err), http.StatusInternalServerError)
		return
	}

	data := IndexPageData{Orders: orders}
	var listBuffer bytes.Buffer
	err = s.template.ExecuteTemplate(&listBuffer, "order_list.html", data.Orders) // Render just the order list part
	if err != nil {
		http.Error(w, fmt.Sprintf("Template execution error for order list: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8") // Important for HTMX swap
	w.WriteHeader(http.StatusOK)
	w.Write(listBuffer.Bytes()) // Write the buffer to the response
}

// handleCancelOrder handles order cancellation requests
func (s *WebServer) handleCancelOrder(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	orderID := vars["id"]

	order, err := s.orderService.GetOrder(orderID)
	if err != nil {
		http.Error(w, fmt.Sprintf("Order not found: %v", err), http.StatusNotFound)
		return
	}

	if order.Status != OrderStatusPending {
		http.Error(w, "Order cannot be cancelled as it is not pending.", http.StatusBadRequest)
		return
	}

	workflowID := order.WorkflowID
	if workflowID == "" {
		http.Error(w, "Workflow ID not associated with order, cannot cancel.", http.StatusInternalServerError) // Should not happen ideally
		return
	}

	// Signal the workflow to cancel
	err = s.temporalClient.SignalWorkflow(r.Context(), workflowID, "", CancelOrderSignalName, nil)
	if err != nil {
		log.Printf("Error signaling workflow %s to cancel: %v", workflowID, err)
		http.Error(w, "Failed to cancel order.", http.StatusInternalServerError)
		return
	}

	// Update order status to cancelled immediately in the service (optimistic update, workflow will confirm)
	err = s.orderService.CancelOrder(orderID) // Update status to cancelled in the service
	if err != nil {
		log.Printf("Error updating order status to cancelled in service: %v", err)
		// Log error but still inform user of cancellation request - workflow is the source of truth
	}

	s.renderOrderList(w, r) // Re-render and send updated order list
}

// handleGetOrder retrieves and displays details for a single order (not directly used in UI yet but good for API)
func (s *WebServer) handleGetOrder(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	orderID := vars["id"]

	order, err := s.orderService.GetOrder(orderID)
	if err != nil {
		http.Error(w, fmt.Sprintf("Order not found: %v", err), http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(order)
}

// handleSSEOrders handles the Server-Sent Events endpoint for order updates
func (s *WebServer) handleSSEOrders(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	log.Println("SSE connection established for orders")
	clientChan := make(chan string) // Channel to send SSE events to client
	go func() {
		defer close(clientChan)
		for {
			// Fetch and render order list
			orders, err := s.orderService.ListOrders()
			if err != nil {
				log.Printf("Error fetching orders for SSE: %v", err)
				// In a real app, consider more robust error handling and backoff
				time.Sleep(5 * time.Second) // Backoff on error
				continue
			}

			data := IndexPageData{Orders: orders}
			var listBuffer bytes.Buffer
			err = s.template.ExecuteTemplate(&listBuffer, "order_list.html", data.Orders) // Just render the order list

			if err != nil {
				log.Printf("Template execution error for SSE: %v", err)
				time.Sleep(5 * time.Second) // Backoff on template error
				continue
			}

			// Send SSE event with rendered HTML
			eventData := fmt.Sprintf("data: %s\n\n", listBuffer.String()) // SSE format: "data: <payload>\n\n"
			clientChan <- eventData

			time.Sleep(3 * time.Second) // Polling interval - adjust as needed
		}
	}()

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming not supported!", http.StatusInternalServerError)
		return
	}

	for msg := range clientChan {
		fmt.Fprint(w, msg) // Send data to client
		flusher.Flush()    // Flush data to client - very important for SSE
	}

	log.Println("SSE connection closed for orders") // Client disconnected or channel closed
}

// compileTemplates parses the HTML templates (moved to main.go, but kept here for reference in web.go if needed)
func compileTemplates() (*template.Template, error) {
	var err error
	funcMap := template.FuncMap{
		"lower": func(s string) string {
			return strings.ToLower(s)
		},
	}

	templates := template.New("").Funcs(funcMap)
	templates, err = templates.ParseGlob("./html/*.html")
	if err != nil {
		return fmt.Errorf("template parsing error: %w", err)
	}

	log.Println(templates.DefinedTemplates())

	return templates, nil
}

// Define a struct to hold data for the index page (moved to web.go as it's web-specific)
type IndexPageData struct {
	Orders []StopLossOrder // Use StopLossOrder struct
}

