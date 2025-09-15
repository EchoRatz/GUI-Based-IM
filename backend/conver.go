package main

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type Member struct {
	UserID primitive.ObjectID `bson:"user_id" json:"user_id"`
	Role   string             `bson:"role" json:"role"`
}

type Conversation struct {
	ID        primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	Title     string             `bson:"title" json:"title"`
	Members   []Member           `bson:"members" json:"members"`
	CreatedAt int64              `bson:"created_at" json:"created_at"`
}

// === Ensure Indexed ===
func ensureConverIndexes(ctx context.Context, db *mongo.Database) error {
	_, err := db.Collection("conversations").Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys: bson.D{{Key: "members.user_id", Value: 1}},
	})
	return err
}

// === Helpers ===
func mustObjectID(hex string) (primitive.ObjectID, error) {
	return primitive.ObjectIDFromHex(hex)
}

func uniqLower(in []string) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(in))
	for _, s := range in {
		u := normalizeUsername(s)
		if u == "" {
			continue
		}
		if _, ok := seen[u]; ok {
			continue
		}
		seen[u] = struct{}{}
		out = append(out, u)
	}
	return out
}

func resolveUsernames(ctx context.Context, db *mongo.Database, usernames []string) ([]primitive.ObjectID, error) {
	if len(usernames) == 0 {
		return nil, errors.New("member list can't be empty")
	}
	cur, err := db.Collection("users").Find(ctx, bson.M{"username": bson.M{"$in": usernames}})
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)

	type udoc struct {
		ID       primitive.ObjectID `bson:"_id"`
		Username string             `bson:"username"`
	}

	ids := make([]primitive.ObjectID, 0, len(usernames))
	found := map[string]struct{}{}
	for cur.Next(ctx) {
		var u udoc
		if err := cur.Decode(&u); err != nil {
			return nil, err
		}
		ids = append(ids, u.ID)
		found[u.Username] = struct{}{}
	}
	for _, name := range usernames {
		if _, ok := found[name]; !ok {
			return nil, errors.New("one or more username do not exist")
		}
	}
	return ids, nil
}

func findExistingDM(ctx context.Context, db *mongo.Database, a, b primitive.ObjectID) (*Conversation, error) {
	filter := bson.M{
		"members": bson.M{"$all": []bson.M{
			{"$elemMatch": bson.M{"user_id": a}},
			{"$elemMatch": bson.M{"user_id": b}},
		}},
		"$expr": bson.M{"$eq": bson.A{bson.M{"$size": "$members"}, 2}},
	}
	var conv Conversation
	err := db.Collection("conversations").FindOne(ctx, filter).Decode(&conv)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return nil, nil
	}
	return &conv, err
}

func getLastMessage(ctx context.Context, db *mongo.Database, cid primitive.ObjectID) (*Message, error) {
	var m Message
	err := db.Collection("messages").FindOne(
		ctx,
		bson.M{"conversation_id": cid},
		options.FindOne().SetSort(bson.D{{Key: "ts", Value: -1}}),
	).Decode(&m)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return nil, nil
	}
	return &m, err
}

// === Handlers ===
// POST / conversations
func CreateConverHandler(client *mongo.Client) gin.HandlerFunc {
	return func(c *gin.Context) {
		uidHex, _ := c.Get("uid")
		uid, err := mustObjectID(uidHex.(string))
		if err != nil {
			c.JSON(401, gin.H{"error": "unauthorized"})
			return
		}

		var in struct {
			Title   string   `json:"title"`
			Members []string `json:"members"`
		}
		if err := c.ShouldBindJSON(&in); err != nil {
			c.JSON(400, gin.H{"error": "bad json"})
			return
		}
		if in.Title == "" {
			in.Title = "New Conversation"
		}

		// ensure creator is included
		creatorUname := c.GetString("uname")
		in.Members = append(in.Members, creatorUname)

		// normalize & req >= 2
		membersU := uniqLower(in.Members)
		if len(membersU) < 2 {
			c.JSON(400, gin.H{"error": "at least 2 unique members required"})
			return
		}

		ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
		defer cancel()
		db := getDB(client)
		_ = ensureConverIndexes(ctx, db)

		memberIDs, err := resolveUsernames(ctx, db, membersU)
		if err != nil {
			c.JSON(400, gin.H{"error": err.Error()})
			return
		}

		// DM Reuse
		if len(memberIDs) == 2 {
			if existing, err := findExistingDM(ctx, db, memberIDs[0], memberIDs[1]); err == nil && existing != nil {
				c.JSON(200, gin.H{
					"id":      existing.ID.Hex(),
					"title":   existing.Title,
					"members": existing.Members,
					"reused":  true,
				})
				return
			}
		}

		// build members with owner role
		members := make([]Member, 0, len(memberIDs))
		for _, id := range memberIDs {
			role := "member"
			if id == uid {
				role = "owner"
			}
			members = append(members, Member{UserID: id, Role: role})
		}

		conv := Conversation{
			Title:     in.Title,
			Members:   members,
			CreatedAt: time.Now().UnixMilli(),
		}

		res, err := db.Collection("conversations").InsertOne(ctx, conv)
		if err != nil {
			fmt.Println("insert conversation error:", err)
			c.JSON(500, gin.H{"error": "db error"})
			return
		}
		c.JSON(201, gin.H{
			"id":      res.InsertedID.(primitive.ObjectID).Hex(),
			"title":   conv.Title,
			"members": conv.Members,
		})
	}
}

// GET /conversations

func ListConverHandler(client *mongo.Client) gin.HandlerFunc {
	type lastMsg struct {
		ID       primitive.ObjectID `json:"id"`
		SenderID primitive.ObjectID `json:"sender_id"`
		Type     string             `json:"type"`
		Body     string             `json:"body"`
		Ts       int64              `json:"ts"`
	}
	type item struct {
		ID        primitive.ObjectID `bson:"_id" json:"id"`
		Title     string             `bson:"title" json:"title"`
		Members   []Member           `bson:"members" json:"members"`
		CreatedAt int64              `bson:"created_at" json:"created_at"`
		Unread    int64              `json:"unread"`
		LastMsg   *lastMsg           `json:"last_msg,omitempty"`
	}

	return func(c *gin.Context) {
		uidHex, _ := c.Get("uid")
		uid, err := primitive.ObjectIDFromHex(uidHex.(string))
		if err != nil {
			c.JSON(401, gin.H{"error": "unauthorized"})
			return
		}

		ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
		defer cancel()
		db := getDB(client)

		// 1. fetch all conver the usr is in
		cur, err := db.Collection("conversations").Find(
			ctx,
			bson.M{"members.user_id": uid},
			options.Find().SetSort(bson.D{{Key: "created_at", Value: -1}}),
		)
		if err != nil {
			c.JSON(500, gin.H{"error": "db error"})
			return
		}
		defer cur.Close(ctx)

		convs := make([]item, 0, 16)
		ids := make([]primitive.ObjectID, 0, 16)
		for cur.Next(ctx) {
			var x item
			if err := cur.Decode(&x); err != nil {
				c.JSON(500, gin.H{"error": "decode error"})
				return
			}
			convs = append(convs, x)
			ids = append(ids, x.ID)
		}

		if len(convs) == 0 {
			c.JSON(200, []item{})
			return
		}

		// 2, load receipt for this user across all those conv -> map[cid]last_read_ts
		recCur, err := db.Collection("receipts").Find(ctx, bson.M{
			"user_id":         uid,
			"conversation_id": bson.M{"$in": ids},
		})
		if err != nil {
			c.JSON(500, gin.H{"error": "db error"})
			return
		}
		type recDoc struct {
			CID        primitive.ObjectID `bson:"conversation_id"`
			LastReadTS int64              `bson:"last_read_ts"`
		}
		lastRead := make(map[primitive.ObjectID]int64, len(ids))
		for recCur.Next(ctx) {
			var r recDoc
			if err := recCur.Decode(&r); err != nil {
				c.JSON(500, gin.H{"error": "decode error"})
				return
			}
			lastRead[r.CID] = r.LastReadTS
		}
		recCur.Close(ctx)

		// 3. for each conver, compute unread + fetch last msg
		for i := range convs {
			cid := convs[i].ID
			// unread
			since := lastRead[cid] // default 0
			n, err := db.Collection("messages").CountDocuments(ctx, bson.M{
				"conversation_id": cid,
				"ts":              bson.M{"$gt": since},
			})
			if err != nil {
				c.JSON(500, gin.H{"error": "db error"})
				return
			}
			convs[i].Unread = n

			// last msg
			if m, err := getLastMessage(ctx, db, cid); err == nil && m != nil {
				convs[i].LastMsg = &lastMsg{
					ID:       m.ID,
					SenderID: m.SenderID,
					Type:     m.Type,
					Body:     m.Body,
					Ts:       m.Ts,
				}
			}
		}

		c.JSON(200, convs)
	}
}
