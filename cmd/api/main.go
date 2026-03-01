package main

import (
	"log"
	"os"

	"gotification/internal/handler"
	"gotification/internal/storage"

	"github.com/gin-gonic/gin"
)

func main() {
	connStr := os.Getenv("DB_CONN")
	if connStr == "" {
		connStr = "postgres://postgres:password@localhost:5432/gotification?sslmode=disable"
	}

	// 1. Initialize DB connection
	db := storage.InitDB(connStr)
	defer db.Close() // Keep DB connection alive until main finishes

	// 2. Setup Data Layer (Repository)
	store := storage.NewNotificationStore(db)

	// 3. Setup HTTP Handler
	h := handler.NewNotificationHandler(store)

	r := gin.Default()

	// 4. Wire routes
	r.POST("/send", h.SendNotification)

	log.Println("Main API server listening on :8080")
	if err := r.Run(":8080"); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
