package main

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	_ "github.com/lib/pq"
)

type NotificationRequest struct {
	UserID  int    `json:"user_id" binding:"required"`
	Title   string `json:"title" binding:"required"`
	Message string `json:"message" binding:"required"`
}

var db *sql.DB

func main() {
	var err error

	connStr := "postgres://postgres:password@localhost:5432/gotification?sslmode=disable"
	db, err = sql.Open("postgres", connStr)
	if err != nil {
		log.Fatalf("Failed to open db connection: %v", err)
	}

	if err = db.Ping(); err != nil {
		log.Fatalf("Cannot connect to postgres: %v", err)
	}
	defer db.Close()

	createTableQuery := `
	CREATE TABLE IF NOT EXISTS notifications (
		id SERIAL PRIMARY KEY,
		user_id INT NOT NULL,
		title VARCHAR(255) NOT NULL,
		message TEXT NOT NULL,
		status VARCHAR(20) NOT NULL,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);`
	if _, err := db.Exec(createTableQuery); err != nil {
		log.Fatalf("Failed to create table: %v", err)
	}

	r := gin.Default()

	r.POST("/send", handleSendNotification)

	log.Println("Main API server listening on :8080")
	if err := r.Run(":8080"); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}

func handleSendNotification(c *gin.Context) {
	var req NotificationRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var notificationID int
	insertQuery := `
		INSERT INTO notifications (user_id, title, message, status)
		VALUES ($1, $2, $3, 'PENDING')
		RETURNING id;
	`
	err := db.QueryRow(insertQuery, req.UserID, req.Title, req.Message).Scan(&notificationID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create notification record"})
		return
	}

	goClient := &http.Client{Timeout: 5 * time.Second}

	bodyBytes, _ := json.Marshal(map[string]interface{}{
		"id":     notificationID,
		"action": "send",
	})

	dummyResponse, err := goClient.Post("http://localhost:8081/process", "application/json", bytes.NewBuffer(bodyBytes))
	if err != nil || dummyResponse.StatusCode != http.StatusOK {
		// Log the error, but we'll mark the db record as FAILED for now instead of returning right away.
		// A more advanced system might retry.
		updateStatus(notificationID, "FAILED")
		c.JSON(http.StatusBadGateway, gin.H{"error": "Dummy server failed to process"})
		return
	}
	defer dummyResponse.Body.Close()

	updateStatus(notificationID, "SENT")

	c.JSON(http.StatusOK, gin.H{
		"message":         "Notification sent successfully",
		"notification_id": notificationID,
		"status":          "SENT",
	})
}

func updateStatus(id int, newStatus string) {
	updateQuery := `UPDATE notifications SET status = $1 WHERE id = $2`
	_, err := db.Exec(updateQuery, newStatus, id)
	if err != nil {
		log.Printf("Failed to update notification %d to %s: %v", id, newStatus, err)
	}
}
