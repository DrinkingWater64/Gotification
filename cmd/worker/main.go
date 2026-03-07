package main

import (
	"bytes"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"gotification/internal/storage"

	"github.com/nats-io/nats.go"
)

type NotificationMessage struct {
	ID     int    `json:"id"`
	Action string `json:"action"`
}

func main() {
	// 1. Initialize NATS connection
	natsURL := os.Getenv("NATS_URL")
	if natsURL == "" {
		natsURL = nats.DefaultURL
	}
	nc, err := nats.Connect(natsURL)
	if err != nil {
		log.Fatalf("Failed to connect to NATS: %v", err)
	}
	defer nc.Close()
	log.Println("Worker connected to NATS successfully.")

	// 2. Initialize DB connection
	connStr := os.Getenv("DB_CONN")
	if connStr == "" {
		connStr = "postgres://postgres:password@localhost:5432/gotification?sslmode=disable"
	}
	db := storage.InitDB(connStr)
	defer db.Close()
	store := storage.NewNotificationStore(db)

	// 3. Prepare Dummy URL
	dummyURL := os.Getenv("DUMMY_URL")
	if dummyURL == "" {
		dummyURL = "http://localhost:8081/process"
	}
	goClient := &http.Client{Timeout: 5 * time.Second}

	// 4. Subscribe to the subject using a Queue Group to load balance messages
	sub, err := nc.QueueSubscribe("notifications.send", "notification_workers", func(m *nats.Msg) {
		var msg NotificationMessage
		if err := json.Unmarshal(m.Data, &msg); err != nil {
			log.Printf("Failed to unmarshal message: %v", err)
			return
		}

		log.Printf("Worker received message to process notification ID: %d", msg.ID)

		// Forward the request to the Dummy Server
		dummyResponse, err := goClient.Post(dummyURL, "application/json", bytes.NewBuffer(m.Data))
		if err != nil || dummyResponse.StatusCode != http.StatusOK {
			log.Printf("Dummy server failed. Marking %d as FAILED.", msg.ID)
			store.UpdateStatus(msg.ID, "FAILED")
			return
		}
		defer dummyResponse.Body.Close()

		// Success! Mark as SENT
		log.Printf("Dummy server succeeded. Marking %d as SENT.", msg.ID)
		store.UpdateStatus(msg.ID, "SENT")
	})

	if err != nil {
		log.Fatalf("Failed to subscribe to NATS: %v", err)
	}
	defer sub.Unsubscribe()

	// 5. Keep the worker running gracefully
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan
	log.Println("Worker shutting down...")
}
