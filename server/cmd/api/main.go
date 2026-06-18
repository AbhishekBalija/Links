package main

import (
	"log"
	"net/http"

	"github.com/AbhishekBalija/Links/server/internal/users"
	"github.com/AbhishekBalija/Links/server/pkg/config"
	"github.com/AbhishekBalija/Links/server/pkg/db"
	"github.com/gin-gonic/gin"
)

func main() {
	// Initialize database
	dsn := config.GetDatabaseDSN()
	if err := db.InitDB(dsn); err != nil {
		log.Fatalf("Database initialization failed: %v", err)
	}

	r := gin.Default()

	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"message": "LINKS API running",
		})
	})

	// User routes
	r.POST("/users", users.CreateUser)
	r.GET("/users", users.GetUsers)
	r.GET("/users/:id", users.GetUserByID)

	r.Run(":8080")
}