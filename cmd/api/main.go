package main

import (
	"log"
	"os"

	"gotification/internal/handler"
	"gotification/internal/storage"

	"github.com/gin-gonic/gin"
	"github.com/nats-io/nats.go"
	ginprometheus "github.com/zsais/go-gin-prometheus"
)

func main() {
	connStr := os.Getenv("DB_CONN")
	if connStr == "" {
		connStr = "postgres://postgres:password@localhost:5432/gotification?sslmode=disable"
	}

	// 1. Initialize DB connection
	db := storage.InitDB(connStr)
	defer db.Close() // Keep DB connection alive until main finishes

	// 2. Initialize NATS connection
	natsURL := os.Getenv("NATS_URL")
	if natsURL == "" {
		natsURL = nats.DefaultURL
	}
	nc, err := nats.Connect(natsURL)
	if err != nil {
		log.Fatalf("Failed to connect to NATS: %v", err)
	}
	defer nc.Close()

	// 3. Setup Data Layer (Repository)
	store := storage.NewNotificationStore(db)

	// 4. Setup HTTP Handler
	h := handler.NewNotificationHandler(store, nc)

	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(gin.Recovery())

	p := ginprometheus.NewPrometheus("gin")
	p.Use(r)

	// 4. Wire routes
	r.POST("/send", h.SendNotification)

	log.Println("Main API server listening on :8080")
	if err := r.Run(":8080"); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
