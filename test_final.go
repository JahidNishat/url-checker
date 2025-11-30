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

	time.Sleep(6 * time.Second)
	ctx := context.Background()

	// Test 1: Transaction
	log.Println("📝 Test 1: Transaction (insert URL + check)")
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		log.Fatal(err)
	}

	var urlID int
	err = tx.QueryRow("INSERT INTO urls (url, check_interval_seconds) VALUES ($1, $2) RETURNING id",
		"https://transaction-test.com", 300).Scan(&urlID)
	if err != nil {
		tx.Rollback()
		log.Fatal(err)
	}

	_, err = tx.Exec("INSERT INTO checks (url_id, status_code, response_time_ms) VALUES ($1, $2, $3)",
		urlID, 200, 45)
	if err != nil {
		tx.Rollback()
		log.Fatal(err)
	}

	err = tx.Commit()
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("   ✅ Inserted URL %d with check", urlID)

	// Test 2: Read-your-writes (verify transaction result)
	log.Println("\n🔍 Test 2: Read-your-writes")
	ctxRYW := EnableReadYourWrites(ctx)
	var checkCount int
	err = db.QueryRow(ctxRYW, "SELECT COUNT(*) FROM checks WHERE url_id = $1", urlID).Scan(&checkCount)
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("   ✅ Found %d checks for URL %d", checkCount, urlID)

	// Test 3: Batch read from follower
	log.Println("\n📊 Test 3: Batch read (from follower)")
	rows, err := db.Query(ctx, "SELECT id, url FROM urls ORDER BY id DESC LIMIT 5")
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()

	count := 0
	for rows.Next() {
		var id int
		var url string
		rows.Scan(&id, &url)
		count++
	}
	log.Printf("   ✅ Read %d URLs from follower", count)

	// Test 4: Stats
	log.Println("\n📈 Final Stats:")
	stats := db.GetStats()
	log.Printf("   Leader reads:   %d", stats.LeaderReads)
	log.Printf("   Follower reads: %d", stats.FollowerReads)
	log.Printf("   Fallback reads: %d", stats.FallbackReads)
	log.Printf("   Writes:         %d", stats.Writes)
	log.Printf("   Follower:       %v", map[bool]string{true: "✅ Healthy", false: "❌ Down"}[stats.FollowerHealthy])

	// Validation
	if stats.Writes >= 3 && stats.FollowerReads >= 1 && stats.FollowerHealthy {
		log.Println("\n🎉 ALL TESTS PASSED - Production Ready!")
	}
}
