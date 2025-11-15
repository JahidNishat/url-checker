package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/redis/go-redis/v9"
)

var (
	httpClient     *http.Client
	processedCount int64
)

func main() {
	//Load config
	config := LoadConfig()

	//Setup HTTP client
	httpClient = &http.Client{
		Timeout: time.Duration(config.HTTPTimeout) * time.Second,
		Transport: &http.Transport{
			MaxIdleConns:        100,
			MaxIdleConnsPerHost: 100,
			IdleConnTimeout:     90 * time.Second,
		},
	}

	workerID := fmt.Sprintf("worker-%d", os.Getpid())
	if len(os.Args) > 1 {
		// Last arg that's not a .go file
		for _, arg := range os.Args[1:] {
			if arg[len(arg)-3:] != ".go" {
				workerID = arg
			}
		}
	}

	rdb := NewRedisClient(config.RedisAddr)
	defer rdb.Close()

	log.Printf("[%s] 🚀 Starting...\n", workerID)

	//Graceful Shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-sigChan
		log.Printf("\n[%s] 🛑 Shutting down gracefully...", workerID)
		log.Printf("[%s] 📊 Processed %d URLs in this session", workerID, atomic.LoadInt64(&processedCount))
		os.Exit(0)
	}()

	maxRetries := config.MaxRetries
	retryCount := 0

	//Main processing loop
	for {
		result, err := rdb.BRPop(ctx, time.Duration(config.WorkerTimeout)*time.Second, "url_queue").Result()
		if err != nil {
			if errors.Is(err, redis.Nil) {
				continue
			}

			log.Printf("[%s] ❌ Redis error: %v\n", workerID, err)
			retryCount++

			if retryCount >= maxRetries {
				log.Printf("[%s] ⚠️  Max retries reached. Exiting.\n", workerID)
				os.Exit(1)
			}

			// Exponential backoff
			backoff := time.Duration(retryCount) * time.Second
			log.Printf("[%s] 🔄 Retrying in %v...", workerID, backoff)
			time.Sleep(backoff)
			continue
		}

		retryCount = 0

		url := result[1]

		rdb.Incr(ctx, "processing")

		urlResult := checkURL(url, workerID, rdb)

		resultJSON, _ := json.Marshal(urlResult)
		rdb.LPush(ctx, "results", resultJSON)

		// Trim results to keep memory under control
		rdb.LTrim(ctx, "results", 0, int64(config.ResultsToKeep-1))

		if urlResult.Error == "" {
			rdb.Incr(ctx, "success")
		} else {
			rdb.Incr(ctx, "error")
		}
		rdb.Decr(ctx, "processing")

		atomic.AddInt64(&processedCount, 1)

		if atomic.LoadInt64(&processedCount)%100 == 0 {
			log.Printf("[%s] 📈 Processed %d URLs", workerID, atomic.LoadInt64(&processedCount))
		}
	}
}

func checkURL(url string, workerID string, rdb *redis.Client) URLResult {
	startTime := time.Now()
	cacheKey := fmt.Sprintf("cache:%s", url)

	cache, err := rdb.Get(ctx, cacheKey).Bytes()
	if err == nil {
		// Cache hit
		rdb.Incr(ctx, "cache_hit")
		cacheRes := URLResult{}
		json.Unmarshal(cache, &cacheRes)
		return cacheRes
	}

	// Cache miss
	rdb.Incr(ctx, "cache_miss")

	result := URLResult{
		URL:       url,
		WorkerID:  workerID,
		CheckedAt: startTime,
	}

	resp, err := httpClient.Get(url)
	if err != nil {
		result.Error = err.Error()
		result.Duration = time.Since(startTime).Milliseconds()
		return result
	}
	defer resp.Body.Close()

	result.Status = resp.StatusCode
	result.Duration = time.Since(startTime).Milliseconds()

	if resp.StatusCode != 200 {
		result.Error = fmt.Sprintf("HTTP %d", resp.StatusCode)
	}

	// Cache result for 5 minutes
	jsonResult, _ := json.Marshal(result)
	rdb.Set(ctx, cacheKey, jsonResult, 5*time.Minute)

	return result
}
