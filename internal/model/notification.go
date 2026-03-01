package model

type NotificationRequest struct {
	UserID  int    `json:"user_id" binding:"required"`
	Title   string `json:"title" binding:"required"`
	Message string `json:"message" binding:"required"`
}
