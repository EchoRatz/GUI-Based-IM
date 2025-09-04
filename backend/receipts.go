package main

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

/*
Schema:
  receipts:
    - conversation_id (ObjectId)
    - user_id        (ObjectId)
    - last_read_ts   (int64, millis)
Unique index on (conversation_id, user_id)
*/

type Receipt struct {
	ID             primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	ConversationID primitive.ObjectID `bson:"conversation_id" json:"conversation_id"`
	UserID         primitive.ObjectID `bson:"user_id" json:"user_id"`
	LastReadTS     int64              `bson:"last_read_ts" json:"last_read_ts"`
}

func ensureReceiptIndexes(ctx context.Context, db *mongo.Database) error {
	_, err := db.Collection("receipts").Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys:    bson.D{{Key: "conversation_id", Value: 1}, {Key: "user_id", Value: 1}},
		Options: options.Index().SetUnique(true),
	})
	return err
}

// POST/conversation/:cid/read
// Body (optional): { "ts": <int64 millis> }
// If ts is omitted, uses now. Only moves forward (never decreases).
func MarkReadHandler(client *mongo.Client) gin.HandlerFunc {
	return func(c *gin.Context) {
		uidHex, _ := c.Get("uid")
		uid, err := mustOID(uidHex.(string))
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}
		cidHex := c.Param("cid")
		cid, err := primitive.ObjectIDFromHex(cidHex)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid conversation id"})
			return
		}

		var in struct {
			Ts *int64 `json:"ts"`
		}
		if err := c.ShouldBindJSON(&in); err != nil {
			// allow empty body
			in.Ts = nil
		}
		now := time.Now().UnixMilli()
		newTs := now
		if in.Ts != nil && *in.Ts > 0 {
			newTs = *in.Ts
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

		if err := ensureReceiptIndexes(ctx, db); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "index error"})
			return
		}

		// upsert and only move forward
		_, err = db.Collection("receipts").UpdateOne(
			ctx,
			bson.M{"conversation_id": cid, "user_id": uid},
			bson.M{
				"$max": bson.M{"last_read_ts": newTs}, // move forward only
				"$setOnInsert": bson.M{
					"conversation_id": cid,
					"user_id":         uid,
				},
			},
			options.Update().SetUpsert(true),
		)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "db error"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"ok": true, "last_read_ts": newTs})
	}
}

// GET /conversations/:cid/unread
// Returns : { unread: <int>, last_read_ts: <int64> }
func UnreadCountHandler(client *mongo.Client) gin.HandlerFunc {
	return func(c *gin.Context) {
		uidHex, _ := c.Get("uid")
		uid, err := mustOID(uidHex.(string))
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}
		cidHex := c.Param("cid")
		cid, err := primitive.ObjectIDFromHex(cidHex)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid conversation id"})
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

		// find last_read_ts (default 0 if none)
		var rc Receipt
		err = db.Collection("receipts").FindOne(ctx,
			bson.M{"conversation_id": cid, "user_id": uid},
		).Decode(&rc)
		var last int64 = 0
		if err == nil {
			last = rc.LastReadTS
		} else if !errors.Is(err, mongo.ErrNoDocuments) {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "db error"})
			return
		}

		// count msg newer than last_read_ts
		n, err := db.Collection("messages").CountDocuments(ctx, bson.M{
			"conversation_id": cid,
			"ts":              bson.M{"$gt": last},
		})
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "db error"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"unread": n, "last_read_ts": last})
	}
}
