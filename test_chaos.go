package main

import (
	"context"
	"log"
	"time"

	_ "github.com/lib/pq"
)

func main() {
	leaderDSN := "postgres://postgres:12345@localhost:5433/distributed_url_checker?sslmode=disable"
	followerDSN := "postgres://postgres:12345@localhost:5434/distributed_url_checker?sslmode=disable"

	db, err := NewDBManager(leaderDSN, followerDSN)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	log.Println("⏳ Waiting for initial health check...")
	time.Sleep(6 * time.Second)

	ctx := context.Background()

	log.Println("🔄 Starting continuous query loop...")
	log.Println("   (Press Ctrl+C to stop)")
	log.Println()

	for i := 1; ; i++ {
		var count int
		err := db.QueryRow(ctx, "SELECT COUNT(*) FROM urls").Scan(&count)

		status := "✅"
		if err != nil {
			status = "❌"
			log.Printf("%s Query #%d FAILED: %v", status, i, err)
		} else {
			log.Printf("%s Query #%d: %d urls | Leader:%d Follower:%d Fallback:%d",
				status, i, count, db.leaderReads, db.followerReads, db.fallbackReads)
		}

		time.Sleep(2 * time.Second)
	}
}
