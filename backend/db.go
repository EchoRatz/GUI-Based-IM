package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// connectMongo connects to MongoDB and pings it to confirm connection.
func connectMongo(uri string) *mongo.Client {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client, err := mongo.Connect(ctx, options.Client().ApplyURI(uri))
	if err != nil {
		panic(err)
	}

	// quick ping
	if err := client.Ping(ctx, nil); err != nil {
		panic(err)
	}

	fmt.Println("✅ Connected to MongoDB")
	return client
}

func getDB(client *mongo.Client) *mongo.Database {
	name := os.Getenv("MONGO_DB")
	if name == "" {
		name = "chatdb" // default
	}
	return client.Database(name)
}
