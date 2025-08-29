package main

import (
	"context"
	"errors"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type User struct {
	ID        primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	Username  string             `bson:"username" json:"username"`
	CreatedAt int64              `bson:"created_at" json:"created_at"`
	LastSeen  int64              `bson:"last_seen" json:"last_seen"`
}

// === Username Rules ===
var (
	usernameRe   = regexp.MustCompile(`^[a-zA-Z0-9_]{3,20}$`)
	reservedName = map[string]struct{}{
		"admin": {}, "root": {}, "system": {}, "null": {}, "undefined": {},
		"me": {}, "you": {}, "owner": {}, "moderator": {}, "support": {},
	}
)

func normalizeUsername(s string) string {
	return strings.ToLower(strings.TrimSpace(s))
}

func validateUsername(u string) error {
	if !usernameRe.MatchString(u) {
		return errors.New("username must match /^[a-zA-Z0-9_]{3,20}$/")
	}
	if _, bad := reservedName[u]; bad {
		return errors.New("username is reserved")
	}
	return nil
}

// === JWT ===

type Claims struct {
	UserID   string `json:"user_id"`
	Username string `json:"username"`
	jwt.RegisteredClaims
}

func jwtSecret() []byte {
	if s := os.Getenv("JWT_SECRET"); s != "" {
		return []byte(s)
	}
	return []byte("dev-secret-key-change-me")
}

func signJWT(id primitive.ObjectID, username string, ttl time.Duration) (string, error) {
	claims := Claims{
		UserID:   id.Hex(),
		Username: username,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(ttl)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	t := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return t.SignedString(jwtSecret())
}

// AuthRequired parses Bearer token and injects uid/uname into context.
func AuthRequired() gin.HandlerFunc {
	return func(c *gin.Context) {
		h := c.GetHeader("Authorization")
		if !strings.HasPrefix(h, "Bearer ") {
			c.AbortWithStatusJSON(401, gin.H{"error": "missing bearer token"})
			return
		}
		tokenStr := strings.TrimPrefix(h, "Bearer ")
		var claims Claims
		_, err := jwt.ParseWithClaims(tokenStr, &claims, func(t *jwt.Token) (interface{}, error) {
			return jwtSecret(), nil
		})
		if err != nil {
			c.AbortWithStatusJSON(401, gin.H{"error": "invalid token"})
			return
		}
		c.Set("uid", claims.UserID)
		c.Set("uname", claims.Username)
		c.Next()
	}
}

// ===== Mongo indexes for users(username unique) =====

func ensureUserIndexes(ctx context.Context, db *mongo.Database) error {
	_, err := db.Collection("users").Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys:    bson.D{{Key: "username", Value: 1}},
		Options: options.Index().SetUnique(true),
	})
	return err
}

// ClaimUsernameHandler implements passwordless "sign-in":
// POST /claim  { "username": "minty_68" }
func ClaimUsernameHandler(client *mongo.Client) gin.HandlerFunc {
	return func(c *gin.Context) {
		var in struct {
			Username string `json:"username"`
		}
		if err := c.ShouldBindJSON(&in); err != nil {
			c.JSON(400, gin.H{"error": "bad json"})
			return
		}

		u := normalizeUsername(in.Username)
		if err := validateUsername(u); err != nil {
			c.JSON(400, gin.H{"error": err.Error()})
			return
		}

		now := time.Now().UnixMilli()
		doc := User{
			Username:  u,
			CreatedAt: now,
			LastSeen:  now,
		}

		ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
		defer cancel()

		db := client.Database("im")
		_ = ensureUserIndexes(ctx, db)

		res, err := db.Collection("users").InsertOne(ctx, doc)
		if mongo.IsDuplicateKeyError(err) {
			// existing user → sign-in
			var existing User
			if err2 := db.Collection("users").FindOne(ctx, bson.M{"username": u}).Decode(&existing); err2 != nil {
				c.JSON(500, gin.H{"error": "db error"})
				return
			}
			_, _ = db.Collection("users").UpdateByID(ctx, existing.ID,
				bson.M{"$set": bson.M{"last_seen": now}})
			tok, _ := signJWT(existing.ID, existing.Username, 24*time.Hour)
			c.JSON(200, gin.H{"token": tok, "user": gin.H{
				"id": existing.ID.Hex(), "username": existing.Username,
			}})
			return
		}
		if err != nil {
			c.JSON(500, gin.H{"error": "db error"})
			return
		}

		oid := res.InsertedID.(primitive.ObjectID)
		tok, _ := signJWT(oid, u, 24*time.Hour)
		c.JSON(201, gin.H{
			"token": tok,
			"user":  gin.H{"id": oid.Hex(), "username": u},
		})
	}
}

// MeHandler: GET /me  (requires AuthRequired)
func MeHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		uid, _ := c.Get("uid")
		uname, _ := c.Get("uname")
		c.JSON(200, gin.H{"user_id": uid, "username": uname})
	}
}
