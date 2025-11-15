package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"time"
)

func main() {
	if len(os.Args) < 2 {
		log.Fatal("Usage: go run producer.go common.go config.go <urls_file>")
	}

	filename := os.Args[len(os.Args)-1]

	// Load config
	config := LoadConfig()

	// Connect to Redis
	rdb := NewRedisClient(config.RedisAddr)
	defer rdb.Close()

	// Test connection
	if err := rdb.Ping(ctx).Err(); err != nil {
		log.Fatal("could not connect to Redis: ", err)
	}

	log.Println("✅ Connected to Redis")

	// Reset counters
	rdb.Set(ctx, "success", 0, 0)
	rdb.Set(ctx, "error", 0, 0)
	rdb.Set(ctx, "processing", 0, 0)
	rdb.Set(ctx, "cache_hit", 0, 0)
	rdb.Set(ctx, "cache_miss", 0, 0)

	// Clear old queue and results
	rdb.Del(ctx, "url_queue")
	rdb.Del(ctx, "results")

	log.Println("🗑️  Cleared previous data")

	// Read and enqueue URLs
	file, err := os.Open(filename)
	if err != nil {
		log.Fatal("could not open file: ", err)
	}
	defer file.Close()

	count := 0
	startTime := time.Now()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		url := scanner.Text()
		if url != "" {
			// Push to queue
			rdb.LPush(ctx, "url_queue", url)
			count++

			// Progress every 10,000 URLs
			if count%10000 == 0 {
				elapsed := time.Since(startTime).Seconds()
				rate := float64(count) / elapsed
				fmt.Printf("\rEnqueued: %d URLs (%.0f URLs/sec)", count, rate)
			}
		}
	}

	if err := scanner.Err(); err != nil {
		log.Fatal("could not read file: ", err)
	}

	// Store total count
	rdb.Set(ctx, "total_urls", count, 0)

	elapsed := time.Since(startTime)
	fmt.Printf("\n\n✅ Enqueued %d URLs in %.2f seconds\n", count, elapsed.Seconds())
	fmt.Printf("📊 Average: %.0f URLs/sec\n", float64(count)/elapsed.Seconds())
	fmt.Println("\n🚀 Ready to start workers!")
}
