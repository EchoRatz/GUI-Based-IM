package main

import (
	"context"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/gorilla/websocket"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

/*
Events pushed to clients:

message.created:
{
  "type": "message.created",
  "conversation_id": "<cid>",
  "payload": {
    "id": "<msgId>",
    "sender_id": "<uid>",
    "type": "text",
    "body": "...",
    "ts": 1712345678901
  }
}

receipt.updated:
{
  "type": "receipt.updated",
  "conversation_id": "<cid>",
  "payload": {
    "user_id": "<uid>",
    "last_read_ts": 1712345678901
  }
}
*/

type Event struct {
	Type           string      `json:"type"`
	ConversationID string      `json:"conversation_id"`
	Payload        interface{} `json:"payload,omitempty"`
}

type wsClient struct {
	conn *websocket.Conn
	send chan Event
	uid  primitive.ObjectID
	cid  primitive.ObjectID
}

type Broadcaster struct {
	mu    sync.RWMutex
	rooms map[primitive.ObjectID]map[*wsClient]struct{}
}

func NewBroadcaster() *Broadcaster {
	return &Broadcaster{
		rooms: make(map[primitive.ObjectID]map[*wsClient]struct{}),
	}
}

func (b *Broadcaster) Join(c *wsClient) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if _, ok := b.rooms[c.cid]; !ok {
		b.rooms[c.cid] = make(map[*wsClient]struct{})
	}
	b.rooms[c.cid][c] = struct{}{}
}

func (b *Broadcaster) Leave(c *wsClient) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if m, ok := b.rooms[c.cid]; ok {
		delete(m, c)
		if len(m) == 0 {
			delete(b.rooms, c.cid)
		}
	}
}

func (b *Broadcaster) Publish(e Event) {
	cid, err := primitive.ObjectIDFromHex(e.ConversationID)
	if err != nil {
		return
	}
	b.mu.RLock()
	defer b.mu.RUnlock()
	m := b.rooms[cid]
	for cl := range m {
		select {
		case cl.send <- e:
		default:
			// client buffer full : drop connection
			go func(cl *wsClient) {
				cl.conn.Close()
			}(cl)
			delete(m, cl)
		}
	}
}

// glocal broadcaster
var broadcaster = NewBroadcaster()

// WS upgrder
var upgrader = websocket.Upgrader{
	// CORS: allow the same origins as your HTTP CORS
	CheckOrigin:      func(r *http.Request) bool { return true },
	HandshakeTimeout: 5 * time.Second,
	ReadBufferSize:   1024,
	WriteBufferSize:  1024,
}

// parse&verify Bearer token
func parseBearer(c *gin.Context) (*Claims, error) {
	h := c.GetHeader("Authorization")
	if len(h) < 8 || h[:7] != "Bearer " {
		return nil, jwt.ErrTokenMalformed
	}
	tok := h[7:]
	var claims Claims
	_, err := jwt.ParseWithClaims(tok, &claims, func(t *jwt.Token) (interface{}, error) {
		return jwtSecret(), nil
	})
	if err != nil {
		return nil, err
	}
	return &claims, nil
}

// add near parseBearer:
func parseBearerOrQuery(c *gin.Context) (*Claims, error) {
	// try Authorization header first
	if cl, err := parseBearer(c); err == nil {
		return cl, nil
	}
	// fallback: ?token=
	tok := c.Query("token")
	if tok == "" {
		return nil, jwt.ErrTokenMalformed
	}
	var claims Claims
	_, err := jwt.ParseWithClaims(tok, &claims, func(t *jwt.Token) (interface{}, error) {
		return jwtSecret(), nil
	})
	if err != nil {
		return nil, err
	}
	return &claims, nil
}

// GET /ws/:cid (Authorization: Bearer <token>)
// Upgrades to WebSocket if the user is a member of conversation
func WSHandler(client *mongo.Client) gin.HandlerFunc {
	return func(c *gin.Context) {
		claims, err := parseBearerOrQuery(c)
		if err != nil {
			c.AbortWithStatus(http.StatusUnauthorized)
			return
		}
		uid, err := primitive.ObjectIDFromHex(claims.UserID)
		if err != nil {
			c.AbortWithStatus(http.StatusUnauthorized)
			return
		}
		cidHex := c.Param("cid")
		cid, err := primitive.ObjectIDFromHex(cidHex)
		if err != nil {
			c.AbortWithStatus(http.StatusBadRequest)
			return
		}

		// membership check
		ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
		defer cancel()
		db := getDB(client)
		ok, err := isMember(ctx, db, cid, uid)
		if err != nil {
			c.AbortWithStatus(http.StatusInternalServerError)
			return
		}
		if !ok {
			c.AbortWithStatus(http.StatusForbidden)
			return
		}

		ws, err := upgrader.Upgrade(c.Writer, c.Request, nil)
		if err != nil {
			return
		}
		cl := &wsClient{
			conn: ws,
			send: make(chan Event, 32),
			uid:  uid,
			cid:  cid,
		}
		broadcaster.Join(cl)

		// writer
		go func() {
			defer func() {
				broadcaster.Leave(cl)
				_ = cl.conn.Close()
			}()
			cl.conn.SetWriteDeadline(time.Now().Add(30 * time.Second))
			for {
				select {
				case e, ok := <-cl.send:
					if !ok {
						return
					}
					cl.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
					if err := cl.conn.WriteJSON(e); err != nil {
						return
					}
				case <-time.After(25 * time.Second):
					// ping to keep alive
					cl.conn.SetWriteDeadline(time.Now().Add(5 * time.Second))
					if err := cl.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
						return
					}
				}
			}
		}()

		// reader
		go func() {
			defer func() {
				broadcaster.Leave(cl)
				_ = cl.conn.Close()
			}()
			for {
				if _, _, err := cl.conn.ReadMessage(); err != nil {
					return
				}
			}
		}()
	}
}
