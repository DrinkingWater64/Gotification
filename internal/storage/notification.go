package storage

import (
	"database/sql"
	"log"
)

// NotificationStore acts as a repository for notifications
type NotificationStore struct {
	db *sql.DB
}

func NewNotificationStore(db *sql.DB) *NotificationStore {
	return &NotificationStore{db: db}
}

func (s *NotificationStore) CreatePending(userID int, title, message string) (int, error) {
	var notificationID int
	insertQuery := `
		INSERT INTO notifications (user_id, title, message, status)
		VALUES ($1, $2, $3, 'PENDING')
		RETURNING id;
	`
	err := s.db.QueryRow(insertQuery, userID, title, message).Scan(&notificationID)
	return notificationID, err
}

func (s *NotificationStore) UpdateStatus(id int, newStatus string) error {
	updateQuery := `UPDATE notifications SET status = $1 WHERE id = $2`
	_, err := s.db.Exec(updateQuery, newStatus, id)
	if err != nil {
		log.Printf("Failed to update notification %d to %s: %v", id, newStatus, err)
	}
	return err
}
