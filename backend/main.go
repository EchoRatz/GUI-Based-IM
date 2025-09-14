package main

import (
	"context"
	"net/http"
	"os"
	"time"

	"github.com/gin-contrib/cors"
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

	// Add CORS middleware
	config := cors.DefaultConfig()
	config.AllowOrigins = []string{"http://localhost:3000", "http://localhost:5173", "http://127.0.0.1:3000", "http://127.0.0.1:5173", "http://localhost:8080", "http://127.0.0.1:8080", "http://127.0.0.1:5500", "https://gui-im.netlify.app/"}
	config.AllowMethods = []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"}
	config.AllowHeaders = []string{"Origin", "Content-Type", "Accept", "Authorization"}
	config.AllowCredentials = true
	r.Use(cors.New(config))

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
	r.GET("/users", AuthRequired(), ListUsersHandler(client))

	// Conversation endpoints
	r.POST("/conversations", AuthRequired(), CreateConverHandler(client))
	r.GET("/conversations", AuthRequired(), ListConverHandler(client))

	// Messages
	r.POST("/messages/:cid", AuthRequired(), SendMessageHandler(client))
	r.GET("/messages/:cid", AuthRequired(), ListMessagesHandler(client))

	// receipts
	r.POST("/conversations/:cid/read", AuthRequired(), MarkReadHandler(client))
	r.GET("/conversations/:cid/unread", AuthRequired(), UnreadCountHandler(client))

	// Local Port
	r.Run(":8080")
}
