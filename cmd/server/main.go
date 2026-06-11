package main

import (
	"context"
	"fmt"
	"log"
	"net/http"

	"ui-prompter/internal/db"
	"ui-prompter/internal/handlers"
	"ui-prompter/internal/middleware"
)

func main() {
	database, err := db.NewDatabase(context.Background(), "file:ui-prompter.db?mode=rwc")
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}

	app := handlers.NewApp(database)
	mux := http.NewServeMux()

	app.RegisterRoutes(mux)

	fmt.Println("Server is starting on port 8080...")
	if err := http.ListenAndServe(":8080", middleware.Logging(mux)); err != nil {
		log.Fatalf("Server failed: %v\n", err)
	}
}
