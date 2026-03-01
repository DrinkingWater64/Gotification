package main

import (
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

func main() {
	r := gin.Default()

	r.POST("/process", func(c *gin.Context) {
		time.Sleep(500 * time.Millisecond)

		c.JSON(http.StatusOK, gin.H{
			"status":  "success",
			"message": "Dummy server processed the notification",
		})
	})

	log.Println("Dummy server is running on port 8081")
	if err := r.Run(":8081"); err != nil {
		log.Fatalf("Failed to run dummy server: %v", err)
	}
}
