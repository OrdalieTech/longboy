package main

import (
	"log"
	"net/http"

	"longboy/api"
	"longboy/internal/config"
	"longboy/internal/database"
	"longboy/internal/models"
)

func main() {
	cfg := config.GetConfig()

	apiDirectory := "./openapi/paypal"
	templateDirectory := "./templates/paypal"
	templates, err := models.LoadAPITemplates(apiDirectory)
	if err != nil {
		log.Fatalf("Failed to load API templates: %v", err)
	}

	err = models.SaveAPITemplates(templates, templateDirectory)
	if err != nil {
		log.Fatalf("Failed to save API templates: %v", err)
	}

	// Initialize database
	db, err := database.InitDB(cfg.GetSecret("DB_PATH"))
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	// Start monitoring active triggers
	go api.MonitorActiveTriggers(db)

	// Set up API routes
	api.SetupRoutes(db)

	// Serve static files from the src directory
	fs := http.FileServer(http.Dir("./src"))
	http.Handle("/static/", http.StripPrefix("/static/", fs))

	// Catch-all route for debugging
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		log.Printf("Received %s request to %s", r.Method, r.URL.Path)
		http.Error(w, "Not found...", http.StatusNotFound)
	})

	log.Println("Server starting on :8080")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatal(err)
	}
}
