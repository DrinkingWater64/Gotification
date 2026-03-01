package handler

import (
	"bytes"
	"encoding/json"
	"net/http"
	"os"
	"time"

	"gotification/internal/model"
	"gotification/internal/storage"

	"github.com/gin-gonic/gin"
)

// NotificationHandler groups together our endpoint logic and dependencies
type NotificationHandler struct {
	store *storage.NotificationStore
}

func NewNotificationHandler(store *storage.NotificationStore) *NotificationHandler {
	return &NotificationHandler{store: store}
}

func (h *NotificationHandler) SendNotification(c *gin.Context) {
	var req model.NotificationRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	notificationID, err := h.store.CreatePending(req.UserID, req.Title, req.Message)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create notification record"})
		return
	}

	goClient := &http.Client{Timeout: 5 * time.Second}

	bodyBytes, _ := json.Marshal(map[string]interface{}{
		"id":     notificationID,
		"action": "send",
	})

	dummyURL := os.Getenv("DUMMY_URL")
	if dummyURL == "" {
		dummyURL = "http://localhost:8081/process"
	}

	dummyResponse, err := goClient.Post(dummyURL, "application/json", bytes.NewBuffer(bodyBytes))
	if err != nil || dummyResponse.StatusCode != http.StatusOK {
		h.store.UpdateStatus(notificationID, "FAILED")
		c.JSON(http.StatusBadGateway, gin.H{"error": "Dummy server failed to process"})
		return
	}
	defer dummyResponse.Body.Close()

	h.store.UpdateStatus(notificationID, "SENT")

	c.JSON(http.StatusOK, gin.H{
		"message":         "Notification sent successfully",
		"notification_id": notificationID,
		"status":          "SENT",
	})
}
