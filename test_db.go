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

	// Wait for health check to stabilize
	log.Println("⏳ Waiting 6 seconds for health check...")
	time.Sleep(6 * time.Second)

	ctx := context.Background()

	// Test 1: Normal read (should go to follower)
	log.Println("\n📖 Test 1: Normal read")
	var count int
	err = db.QueryRow(ctx, "SELECT COUNT(*) FROM urls").Scan(&count)
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("   URL count: %d", count)

	// Test 2: Write to leader
	log.Println("\n✍️  Test 2: Write")
	_, err = db.Exec(ctx, "INSERT INTO urls (url, check_interval_seconds) VALUES ($1, $2)",
		"https://routing-test.com", 300)
	if err != nil {
		log.Fatal(err)
	}
	log.Println("   Inserted new URL")

	// Test 3: Read-your-writes (should go to leader)
	log.Println("\n🔄 Test 3: Read-your-writes")
	ctxRYW := EnableReadYourWrites(ctx)
	var id int
	err = db.QueryRow(ctxRYW, "SELECT id FROM urls WHERE url = $1",
		"https://routing-test.com").Scan(&id)
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("   Found URL with ID: %d", id)

	// Print routing stats
	log.Println("\n📊 Routing Stats:")
	log.Printf("   Leader reads:   %d (expected: 1)", db.leaderReads)
	log.Printf("   Follower reads: %d (expected: 1)", db.followerReads)
	log.Printf("   Fallback reads: %d (expected: 0)", db.fallbackReads)
	log.Printf("   Writes:         %d (expected: 1)", db.writes)

	if db.followerReads == 1 && db.leaderReads == 1 && db.writes == 1 {
		log.Println("\n✅ ALL TESTS PASSED!")
	} else {
		log.Println("\n❌ Unexpected routing behavior")
	}
}
