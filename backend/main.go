package main

import (
	"context"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
)

func main() {
	mongoURI := os.Getenv("MONGO_URI")
	if mongoURI == "" {
		mongoURI = "mongodb://localhost:27017"
	}

	client := connectMongo(mongoURI)
	defer client.Disconnect(context.Background())

	r := gin.Default()
	r.SetTrustedProxies(nil) // remove warning

	// simple ping
	r.GET("/ping", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"msg": "pong"})
	})

	// ✅ add db health check
	r.GET("/health/db", func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		if err := client.Ping(ctx, nil); err != nil {
			c.JSON(500, gin.H{"ok": false, "err": err.Error()})
			return
		}
		c.JSON(200, gin.H{"ok": true})
	})

	// 🔐 auth (must be present)
	r.POST("/claim", ClaimUsernameHandler(client))
	r.GET("/me", AuthRequired(), MeHandler())

	// Conver
	r.POST("/conversations", AuthRequired(), CreateConverHandler(client))
	r.GET("/conversations", AuthRequired(), ListConverHandler(client))

	r.Run(":8080") //OLOLOLOLLO
}
