package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"math/rand"
	"time"

	_ "github.com/lib/pq"
)

func main() {
	// Connect to Postgres
	db, err := sql.Open("postgres", "host=localhost port=5432 user=postgres password=12345 dbname=urlchecker sslmode=disable")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	ctx := context.Background()
	rand.Seed(time.Now().UnixNano())

	log.Println("🗑️  Clearing existing data...")
	db.Exec("TRUNCATE urls, checks, tags, check_tags CASCADE")

	// Step 1: Insert URLs
	log.Println("📝 Inserting URLs...")
	urls := []struct {
		url    string
		domain string
	}{
		{"https://www.google.com", "google.com"},
		{"https://www.github.com", "github.com"},
		{"https://api.stripe.com/v1/charges", "stripe.com"},
		{"https://www.amazon.com", "amazon.com"},
		{"https://www.reddit.com", "reddit.com"},
		{"https://api.twitter.com/2/tweets", "twitter.com"},
		{"https://www.stackoverflow.com", "stackoverflow.com"},
		{"https://www.youtube.com", "youtube.com"},
		{"https://api.github.com/repos", "github.com"},
		{"https://www.facebook.com", "facebook.com"},
		{"https://api.openai.com/v1/chat", "openai.com"},
		{"https://www.netflix.com", "netflix.com"},
		{"https://api.slack.com/api/chat.postMessage", "slack.com"},
		{"https://www.linkedin.com", "linkedin.com"},
		{"https://www.shopify.com", "shopify.com"},
		{"https://api.twilio.com/2010-04-01/Accounts", "twilio.com"},
		{"https://www.dropbox.com", "dropbox.com"},
		{"https://www.notion.so", "notion.so"},
		{"https://api.sendgrid.com/v3/mail/send", "sendgrid.com"},
		{"https://www.cloudflare.com", "cloudflare.com"},
	}

	urlIDs := make([]int, len(urls))
	for i, u := range urls {
		err := db.QueryRowContext(ctx,
			"INSERT INTO urls (url, domain) VALUES ($1, $2) RETURNING id",
			u.url, u.domain,
		).Scan(&urlIDs[i])
		if err != nil {
			log.Fatal(err)
		}
	}
	log.Printf("✅ Inserted %d URLs\n", len(urls))

	// Step 2: Insert tags
	log.Println("📝 Inserting tags...")
	tags := []string{"production", "staging", "critical", "api"}
	tagIDs := make([]int, len(tags))
	for i, tag := range tags {
		err := db.QueryRowContext(ctx,
			"INSERT INTO tags (name) VALUES ($1) RETURNING id",
			tag,
		).Scan(&tagIDs[i])
		if err != nil {
			log.Fatal(err)
		}
	}
	log.Printf("✅ Inserted %d tags\n", len(tags))

	// Step 3: Insert checks (10,000 total, spread over 90 days)
	log.Println("📝 Inserting 10,000 checks...")
	checksPerURL := 500
	now := time.Now()
	start := time.Now()

	checkIDs := make([]int, 0, 10000)

	for _, urlID := range urlIDs {
		for j := 0; j < checksPerURL; j++ {
			// Random timestamp in last 90 days
			daysAgo := rand.Intn(90)
			hoursAgo := rand.Intn(24)
			minutesAgo := rand.Intn(60)
			checkedAt := now.AddDate(0, 0, -daysAgo).Add(-time.Duration(hoursAgo) * time.Hour).Add(-time.Duration(minutesAgo) * time.Minute)

			// Random status (80% success, 15% 404, 5% 500)
			var status int
			var errorMsg *string
			roll := rand.Intn(100)
			if roll < 80 {
				status = 200
			} else if roll < 95 {
				status = 404
				msg := "Not Found"
				errorMsg = &msg
			} else {
				status = 500
				msg := "Internal Server Error"
				errorMsg = &msg
			}

			// Random duration (10-500ms for success, 1000-5000ms for errors)
			var duration int
			if status == 200 {
				duration = 10 + rand.Intn(490)
			} else {
				duration = 1000 + rand.Intn(4000)
			}

			// Random worker
			workerID := fmt.Sprintf("worker-%d", rand.Intn(5)+1)

			var checkID int
			err := db.QueryRowContext(ctx,
				"INSERT INTO checks (url_id, status, duration_ms, checked_at, worker_id, error_msg) VALUES ($1, $2, $3, $4, $5, $6) RETURNING id",
				urlID, status, duration, checkedAt, workerID, errorMsg,
			).Scan(&checkID)
			if err != nil {
				log.Fatal(err)
			}

			checkIDs = append(checkIDs, checkID)
		}
	}
	elapsed := time.Since(start)
	log.Printf("✅ Inserted 10,000 checks in %v (%.0f checks/sec)\n", elapsed, 10000/elapsed.Seconds())

	// Step 4: Link checks to tags (each check gets 1-3 random tags)
	log.Println("📝 Linking checks to tags...")
	start = time.Now()
	for _, checkID := range checkIDs {
		numTags := 1 + rand.Intn(3) // 1-3 tags per check
		usedTags := make(map[int]bool)

		for i := 0; i < numTags; i++ {
			tagID := tagIDs[rand.Intn(len(tagIDs))]
			if usedTags[tagID] {
				continue // Skip duplicate tags on same check
			}
			usedTags[tagID] = true

			_, err := db.ExecContext(ctx,
				"INSERT INTO check_tags (check_id, tag_id) VALUES ($1, $2)",
				checkID, tagID,
			)
			if err != nil {
				log.Fatal(err)
			}
		}
	}
	elapsed = time.Since(start)
	log.Printf("✅ Linked checks to tags in %v\n", elapsed)

	// Print summary
	log.Println("\n" + "════════════════════════════════════════")
	log.Println("📊 DATABASE SUMMARY")
	log.Println("════════════════════════════════════════")

	var urlCount, checkCount, tagCount, linkCount int
	db.QueryRow("SELECT COUNT(*) FROM urls").Scan(&urlCount)
	db.QueryRow("SELECT COUNT(*) FROM checks").Scan(&checkCount)
	db.QueryRow("SELECT COUNT(*) FROM tags").Scan(&tagCount)
	db.QueryRow("SELECT COUNT(*) FROM check_tags").Scan(&linkCount)

	log.Printf("URLs:        %d\n", urlCount)
	log.Printf("Checks:      %d\n", checkCount)
	log.Printf("Tags:        %d\n", tagCount)
	log.Printf("Tag links:   %d\n", linkCount)

	// Count by status
	log.Println("\nChecks by status:")
	rows, _ := db.Query("SELECT status, COUNT(*) FROM checks GROUP BY status ORDER BY status")
	for rows.Next() {
		var status, count int
		rows.Scan(&status, &count)
		log.Printf("  %d: %d (%.1f%%)\n", status, count, float64(count)*100/float64(checkCount))
	}

	log.Println("════════════════════════════════════════")
	log.Println("✅ Ready to test queries!")
}
