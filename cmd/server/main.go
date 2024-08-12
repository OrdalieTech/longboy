package main

import (
	"log"
	"net/http"

	"longboy/api"
	"longboy/internal/database"
)

func main() {
	// Initialize database
	db, err := database.InitDB("./db/longboy.db")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

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
