package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
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

	// Insert a URL check result (POSTGRES WAY: multiple INSERTs with JOINs)
	start := time.Now()

	// Step 1: Insert URL (or get existing ID)
	var urlID int
	err = db.QueryRowContext(ctx,
		"INSERT INTO urls (url, domain) VALUES ($1, $2) ON CONFLICT (url) DO UPDATE SET url = EXCLUDED.url RETURNING id",
		"https://www.google.com", "google.com",
	).Scan(&urlID)
	if err != nil {
		log.Fatal(err)
	}

	// Step 2: Insert check result
	var checkID int
	err = db.QueryRowContext(ctx,
		"INSERT INTO checks (url_id, status, duration_ms, worker_id, error_msg) VALUES ($1, $2, $3, $4, $5) RETURNING id",
		urlID, 200, 150, "worker-1", nil,
	).Scan(&checkID)
	if err != nil {
		log.Fatal(err)
	}

	// Step 3: Insert tags (or get existing IDs)
	tags := []string{"production", "critical"}
	for _, tagName := range tags {
		var tagID int
		err = db.QueryRowContext(ctx,
			"INSERT INTO tags (name) VALUES ($1) ON CONFLICT (name) DO UPDATE SET name = EXCLUDED.name RETURNING id",
			tagName,
		).Scan(&tagID)
		if err != nil {
			log.Fatal(err)
		}

		// Step 4: Link check to tag
		_, err = db.ExecContext(ctx,
			"INSERT INTO check_tags (check_id, tag_id) VALUES ($1, $2) ON CONFLICT DO NOTHING",
			checkID, tagID,
		)
		if err != nil {
			log.Fatal(err)
		}
	}

	elapsed := time.Since(start)
	fmt.Printf("✅ Inserted check in Postgres: %v\n", elapsed)
	fmt.Printf("   URL ID: %d, Check ID: %d\n", urlID, checkID)
}
