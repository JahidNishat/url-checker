package main

import (
	"fmt"
	"log"
	"time"
)

func main() {
	//Load Config
	config := LoadConfig()

	//Connect to Redis
	rdb := NewRedisClient(config.RedisAddr)
	defer rdb.Close()

	log.Println("📊 Real-time Monitor Started")
	log.Println("Press Ctrl+C to stop")
	log.Println()

	startTime := time.Now()
	lastCompleted := int64(0)

	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		stats := GetStats(rdb)
		cacheHits, _ := rdb.Get(ctx, "cache_hit").Int64()
		cacheMisses, _ := rdb.Get(ctx, "cache_miss").Int64()

		completed := stats.Success + stats.Error
		elapsed := time.Since(startTime).Seconds()

		overallRate := float64(completed) / elapsed
		currentRate := float64(int64(completed) - lastCompleted)

		remaining := stats.QueueLength - stats.Processing
		var eta string
		if currentRate > 0 {
			etaSeconds := float64(remaining) / currentRate
			eta = formatDuration(time.Duration(etaSeconds * float64(time.Second)))
		} else {
			eta = "calculating..."
		}

		total := stats.Total
		var progress float64
		if total > 0 {
			progress = float64(completed) / float64(total) * 100
		}

		var hitRate float64
		totalCacheChecks := cacheHits + cacheMisses
		if totalCacheChecks > 0 {
			hitRate = (float64(cacheHits) / float64(totalCacheChecks)) * 100
		}

		// Display
		fmt.Printf("\r\033[K") // Clear line
		fmt.Printf("📊 Queue: %6d | ⚙️  Processing: %3d | ✅ Success: %8d | ❌ Error: %8d | Progress: %.1f%% | Rate: %.0f/s | ETA: %s | 🎯 Cache Hit: %.1f%% (%d hits)",
			stats.QueueLength,
			stats.Processing,
			stats.Success,
			stats.Error,
			progress,
			currentRate,
			eta,
			hitRate,
			cacheHits,
		)

		// Check if done
		if stats.QueueLength == 0 && stats.Processing == 0 && completed > 0 {
			fmt.Println("\n\n🎉 ALL DONE!")
			fmt.Printf("✅ Success: %d\n", stats.Success)
			fmt.Printf("❌ Errors: %d\n", stats.Error)
			fmt.Printf("⏱️  Total Time: %s\n", formatDuration(time.Since(startTime)))
			fmt.Printf("📈 Average Rate: %.0f URLs/sec\n", overallRate)
			break
		}

		lastCompleted = int64(completed)

	}
}

func formatDuration(d time.Duration) string {
	h := int(d.Hours())
	m := int(d.Minutes()) % 60
	s := int(d.Seconds()) % 60

	if h > 0 {
		return fmt.Sprintf("%dh %dm %ds", h, m, s)
	} else if m > 0 {
		return fmt.Sprintf("%dm %ds", m, s)
	} else {
		return fmt.Sprintf("%ds", s)
	}
}
