package main

import (
	"fmt"
	"html/template"
	"log"
	"net/http"
	"strings"
)

// Define a struct to hold data for the index page
type IndexPageData struct {
	Orders []Order // Placeholder for Order data (will be populated later)
}

// Global templates variable to store parsed templates
var templates *template.Template

// indexHandler handles the root path and renders the index page
func indexHandler(w http.ResponseWriter, r *http.Request) {

	log.Println("loading index")

	data := IndexPageData{
		Orders: []Order{}, // Initially empty order list
	}
	err := templates.ExecuteTemplate(w, "layout.html", data)
	if err != nil {
		http.Error(w, fmt.Sprintf("Template execution error: %v", err), http.StatusInternalServerError)
	}
}

// sseOrdersHandler is a placeholder for SSE endpoint
func sseOrdersHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("SSE connection requested")           // Just log for now, implement SSE logic later
	fmt.Fprint(w, "SSE endpoint - to be implemented") // Simple response for now
}

// SetupRoutes configures the HTTP routes for the application.
func SetupRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/", indexHandler)
	mux.HandleFunc("/sse-orders", sseOrdersHandler) // Route for SSE endpoint
}

// compileTemplates parses the HTML templates
func compileTemplates() error {
	var err error
	funcMap := template.FuncMap{
		"lower": func(s string) string {
			return strings.ToLower(s)
		},
	}

	templates = template.New("").Funcs(funcMap)
	templates, err = templates.ParseGlob("./html/*.html")
	if err != nil {
		return fmt.Errorf("template parsing error: %w", err)
	}

	log.Println(templates.DefinedTemplates())

	return nil
}

