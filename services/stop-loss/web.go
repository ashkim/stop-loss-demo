package main

import (
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

type WebServer struct {
	template             *template.Template
	orderWorkflowService OrderWorkflowService
	ordersRepo           OrdersRepo
}

func NewWebServer(tpl *template.Template, tc client.Client, repo OrdersRepo, orderWorkflowService OrderWorkflowService) *WebServer {
	return &WebServer{
		template:             tpl,
		ordersRepo:           repo,
		orderWorkflowService: orderWorkflowService,
	}
}

func (s *WebServer) SetupRoutes(mux *mux.Router) {
	mux.HandleFunc("/", s.handleIndex).Methods("GET")
	mux.HandleFunc("/orders", s.handleCreateOrder).Methods("POST")
	mux.HandleFunc("/orders", s.handleGetOrders).Methods("GET")
	mux.HandleFunc("/orders/{id}/cancel", s.handleCancelOrder).Methods("POST")
}

// handleIndex renders the main index page
func (s *WebServer) handleIndex(w http.ResponseWriter, r *http.Request) {
	log.Println("Loading index page")
	orders, err := s.ordersRepo.ListOrders()
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
		Status:    OrderStatusPending,
	}

	err = s.orderWorkflowService.CreateOrder(r.Context(), order)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to create order: %v", err), http.StatusInternalServerError)
		return
	}
}

func (s *WebServer) handleGetOrders(w http.ResponseWriter, r *http.Request) {
	orders, err := s.ordersRepo.ListOrders()
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to load orders: %v", err), http.StatusInternalServerError)
		return
	}
	err = s.template.ExecuteTemplate(w, "orders_list.html", orders)
	if err != nil {
		http.Error(w, fmt.Sprintf("Template execution error: %v", err), http.StatusInternalServerError)
	}
}

// handleCancelOrder handles order cancellation requests
func (s *WebServer) handleCancelOrder(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	orderID := vars["id"]

	order, err := s.ordersRepo.GetOrder(orderID)
	if err != nil {
		http.Error(w, fmt.Sprintf("Order not found: %v", err), http.StatusNotFound)
		return
	}

	// TODO: probably a race condition here
	if order.Status != OrderStatusPending {
		http.Error(w, "Order cannot be cancelled as it is not pending.", http.StatusBadRequest)
		return
	}

	workflowID := order.WorkflowID
	if workflowID == "" {
		http.Error(w, "Workflow ID not associated with order, cannot cancel.", http.StatusInternalServerError) // Should not happen ideally
		return
	}

	err = s.orderWorkflowService.CancelOrder(r.Context(), workflowID)
	if err != nil {
		log.Printf("Web: Error signaling workflow %s to cancel: %v", workflowID, err)
		http.Error(w, "Failed to cancel order.", http.StatusInternalServerError)
		return
	}
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
		return nil, fmt.Errorf("template parsing error: %w", err)
	}

	log.Println(templates.DefinedTemplates())

	return templates, nil
}

// Define a struct to hold data for the index page (moved to web.go as it's web-specific)
type IndexPageData struct {
	Orders []StopLossOrder // Use StopLossOrder struct
}
