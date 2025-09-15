package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// === Models ===
// --- Model: fix field name (optional but recommended)
type Message struct {
	ID             primitive.ObjectID `bson:"_id,omitempty"   json:"id"`
	ConversationID primitive.ObjectID `bson:"conversation_id" json:"conversation_id"` // <- renamed
	SenderID       primitive.ObjectID `bson:"sender_id"       json:"sender_id"`
	Type           string             `bson:"type"            json:"type"`
	Body           string             `bson:"body"            json:"body"`
	Ts             int64              `bson:"ts"              json:"ts"`
}

// === Indexes ===

func ensureMsgIndexes(ctx context.Context, db *mongo.Database) error {
	c := db.Collection("messages")
	// 1. by conversation (ts desc) for fast timeline reads
	if _, err := c.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys: bson.D{{Key: "conversation_id", Value: 1}, {Key: "ts", Value: -1}},
	}); err != nil {
		return err
	}
	// 2. basic sender filter if ever need it
	_, _ = c.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys: bson.D{{Key: "sender_id", Value: 1}},
	})
	return nil
}

// === Helpers ===
func mustOID(hex string) (primitive.ObjectID, error) {
	return primitive.ObjectIDFromHex(hex)
}

// check if uid is in the conversation's member
func isMember(ctx context.Context, db *mongo.Database, cid, uid primitive.ObjectID) (bool, error) {
	filter := bson.M{
		"_id":             cid,
		"members.user_id": uid,
	}
	var x struct {
		ID primitive.ObjectID `bson:"_id"`
	}
	err := db.Collection("conversations").FindOne(ctx, filter).Decode(&x)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return false, nil
	}
	return err == nil, err
}

// === Handlers ===
// POST /messages/:cid
// Body: { "type": "text", "body": "Hello" }
func SendMessageHandler(client *mongo.Client) gin.HandlerFunc {
	return func(c *gin.Context) {
		uidHex, _ := c.Get("uid")
		uid, err := mustOID(uidHex.(string))
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}

		cidHex := c.Param("cid")
		cid, err := mustOID(cidHex)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid conversation id"})
			return
		}

		var in struct {
			Type string `json:"type"`
			Body string `json:"body"`
		}
		if err := c.ShouldBindJSON(&in); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "bad json"})
			return
		}
		if in.Type == "" {
			in.Type = "text"
		}
		// minimal validation
		if in.Type != "text" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "unsupported message type"})
			return
		}
		if l := len(in.Body); l == 0 || l > 2048 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "body must be 1-2048 chars"})
			return
		}

		ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
		defer cancel()
		db := getDB(client)

		ok, err := isMember(ctx, db, cid, uid)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "db error"})
			return
		}
		if !ok {
			c.JSON(http.StatusForbidden, gin.H{"error": "not a member"})
			return
		}

		if err := ensureMsgIndexes(ctx, db); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "index error"})
			return
		}

		msg := Message{
			ConversationID: cid,
			SenderID:       uid,
			Type:           in.Type,
			Body:           in.Body,
			Ts:             time.Now().UnixMilli(),
		}
		res, err := db.Collection("messages").InsertOne(ctx, msg)
		if err != nil {
			fmt.Println("insert message error:", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "db error"})
			return
		}
		msg.ID = res.InsertedID.(primitive.ObjectID)

		// boradcast to connected clients in this conversation
		broadcaster.Publish(Event{
			Type:           "message.created",
			ConversationID: cid.Hex(),
			Payload: gin.H{
				"id":        msg.ID.Hex(),
				"sender_id": uid.Hex(),
				"type":      msg.Type,
				"body":      msg.Body,
				"ts":        msg.Ts,
			},
		})
		c.JSON(http.StatusCreated, msg)
	}
}

// GET/messages/:cid?before=<ts>&limit=50
// Returns newest -> oldest (reverse-chronological)
func ListMessagesHandler(client *mongo.Client) gin.HandlerFunc {
	return func(c *gin.Context) {
		// auth & params
		uidHex, _ := c.Get("uid")
		uid, err := mustOID(uidHex.(string))
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}
		cidHex := c.Param("cid")
		cid, err := mustOID(cidHex)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid conversation id"})
			return
		}

		// pagination
		limit := 50
		if s := c.Query("limit"); s != "" {
			if n, err := strconv.Atoi(s); err == nil && n > 0 {
				if n > 200 {
					n = 200
				}
				limit = n
			}
		}

		// NEW: since (optional) — fetch only newer than this ts
		var since *int64
		if s := c.Query("since"); s != "" {
			if n, err := strconv.ParseInt(s, 10, 64); err == nil && n > 0 {
				since = &n
			}
		}

		// existing: before (for reverse-chron paging)
		var before int64 = time.Now().UnixMilli() + 1
		if s := c.Query("before"); s != "" {
			if n, err := strconv.ParseInt(s, 10, 64); err == nil && n > 0 {
				before = n
			}
		}

		ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
		defer cancel()
		db := getDB(client)

		// membership gate
		ok, err := isMember(ctx, db, cid, uid)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "db error"})
			return
		}
		if !ok {
			c.JSON(http.StatusForbidden, gin.H{"error": "not a member"})
			return
		}

		// Build filter:
		// - if since provided, use ts > since (to get *new* messages)
		// - else use ts < before (your original reverse-chron window)
		filter := bson.M{"conversation_id": cid}
		if since != nil {
			filter["ts"] = bson.M{"$gt": *since}
		} else {
			filter["ts"] = bson.M{"$lt": before}
		}

		cur, err := db.Collection("messages").Find(
			ctx,
			filter,
			options.Find().
				SetSort(bson.D{{Key: "ts", Value: -1}}).
				SetLimit(int64(limit)),
		)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "db error"})
			return
		}
		defer cur.Close(ctx)

		out := make([]Message, 0, limit)
		for cur.Next(ctx) {
			var m Message
			if err := cur.Decode(&m); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "decode error"})
				return
			}
			out = append(out, m)
		}
		c.JSON(http.StatusOK, out)
	}
}
