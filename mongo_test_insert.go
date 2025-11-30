package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type CheckResult struct {
	Status     int       `bson:"status"`
	DurationMs int       `bson:"duration_ms"`
	CheckedAt  time.Time `bson:"checked_at"`
	WorkerID   string    `bson:"worker_id"`
	ErrorMsg   *string   `bson:"error_msg,omitempty"`
	Tags       []string  `bson:"tags"`
}

type URLDocument struct {
	URL    string        `bson:"url"`
	Domain string        `bson:"domain"`
	Checks []CheckResult `bson:"checks"`
}

func main() {
	// Connect to MongoDB
	client, err := mongo.Connect(context.Background(), options.Client().ApplyURI("mongodb://localhost:27017"))
	if err != nil {
		log.Fatal(err)
	}
	defer client.Disconnect(context.Background())

	coll := client.Database("urlchecker").Collection("urls")
	ctx := context.Background()

	// Insert a URL check result (MONGODB WAY: one upsert with embedded document)
	start := time.Now()

	newCheck := CheckResult{
		Status:     200,
		DurationMs: 150,
		CheckedAt:  time.Now(),
		WorkerID:   "worker-1",
		Tags:       []string{"production", "critical"},
	}

	// Upsert: If URL exists, append to checks array; if not, create document
	filter := bson.M{"url": "https://www.google.com"}
	update := bson.M{
		"$setOnInsert": bson.M{
			"domain": "google.com",
		},
		"$push": bson.M{
			"checks": newCheck,
		},
	}

	_, err = coll.UpdateOne(ctx, filter, update, options.Update().SetUpsert(true))
	if err != nil {
		log.Fatal(err)
	}

	elapsed := time.Since(start)
	fmt.Printf("✅ Inserted check in MongoDB: %v\n", elapsed)
}
