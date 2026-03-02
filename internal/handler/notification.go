package handler

import (
	"encoding/json"
	"net/http"

	"gotification/internal/model"
	"gotification/internal/storage"

	"github.com/gin-gonic/gin"
	"github.com/nats-io/nats.go"
)

// NotificationHandler groups together our endpoint logic and dependencies
type NotificationHandler struct {
	store *storage.NotificationStore
	nc    *nats.Conn
}

func NewNotificationHandler(store *storage.NotificationStore, nc *nats.Conn) *NotificationHandler {
	return &NotificationHandler{store: store, nc: nc}
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

	// In Phase 2, we just publish to the queue and let the worker handle it!
	bodyBytes, _ := json.Marshal(map[string]interface{}{
		"id":     notificationID,
		"action": "send",
	})

	if err := h.nc.Publish("notifications.send", bodyBytes); err != nil {
		// If publishing fails, we can either fail the request or mark DB as FAILED.
		h.store.UpdateStatus(notificationID, "FAILED")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to queue notification"})
		return
	}

	c.JSON(http.StatusAccepted, gin.H{
		"message":         "Notification queued successfully",
		"notification_id": notificationID,
		"status":          "PENDING_WORKER",
	})
}
