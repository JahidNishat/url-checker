package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/redis/go-redis/v9"
)

var (
	httpClient     *http.Client
	processedCount int64
	stampede       *StampedePreventer
	cacheManager   *CacheManager
	err            error
)

type ResultsFlusher struct {
	rdb         *redis.Client
	resultsChan chan URLResult
	stopChan    chan struct{}
	wg          sync.WaitGroup
}

func NewResultsFlusher(rdb *redis.Client) *ResultsFlusher {
	f := &ResultsFlusher{
		rdb:         rdb,
		resultsChan: make(chan URLResult, 1000),
		stopChan:    make(chan struct{}),
	}
	f.wg.Add(1)
	go f.run()
	return f
}

func (f *ResultsFlusher) Add(ctx context.Context, result URLResult) {
	select {
	case f.resultsChan <- result:
	//Buffer successfully
	default:
		data, _ := json.Marshal(result)
		f.rdb.LPush(ctx, "results", data)
		log.Println("⚠️ Flusher channel full, direct write fallback")
	}
}

func (f *ResultsFlusher) run() {
	defer f.wg.Done()

	batch := make([]URLResult, 0, 500)
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	flush := func() {
		if len(batch) == 0 {
			return
		}

		args := make([]interface{}, len(batch))
		for i, result := range batch {
			data, _ := json.Marshal(result)
			args[i] = data
		}

		ctx := context.Background()
		if err := f.rdb.LPush(ctx, "results", args...).Err(); err != nil {
			log.Printf("❌ Flush failed: %v\n", err)
		} else {
			log.Printf("📦 Flushed %d results to Redis\n", len(batch))
		}

		batch = batch[:0]
	}

	for {
		select {
		case result := <-f.resultsChan:
			batch = append(batch, result)

			if len(batch) >= 500 {
				flush()
			}

		case <-ticker.C:
			flush()
		case <-f.stopChan:
			flush()
			return
		}
	}
}

func (f *ResultsFlusher) Stop() {
	close(f.stopChan)
	f.wg.Wait()
	close(f.resultsChan)
}

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

	flusher := NewResultsFlusher(rdb)
	defer flusher.Stop()

	stampede = NewStampedePreventer()

	cacheManager, err = NewCacheManager(1000, rdb, stampede)
	if err != nil {
		log.Printf("[%s] ❌ failed to create cache manager: %v\n", workerID, err)
		os.Exit(1)
	}

	log.Printf("[%s] 🚀 Starting...\n", workerID)

	//Graceful Shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-sigChan
		log.Printf("\n[%s] 🛑 Shutting down gracefully...", workerID)
		log.Printf("[%s] 📊 Processed %d URLs in this session", workerID, atomic.LoadInt64(&processedCount))

		// ✅ NEW: Flush remaining batch before exit
		flusher.Stop()
		log.Printf("[%s] ✅ All batches flushed", workerID)

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

		flusher.Add(ctx, urlResult)

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
	return cacheManager.Get(ctx, url, func(u string) URLResult {
		fetchStart := time.Now()
		res := URLResult{
			URL:       u,
			WorkerID:  workerID,
			CheckedAt: fetchStart,
		}

		resp, err := httpClient.Get(u)
		if err != nil {
			res.Error = err.Error()
			res.Duration = time.Since(fetchStart).Milliseconds()
			return res
		}
		defer resp.Body.Close()

		res.Status = resp.StatusCode
		res.Duration = time.Since(fetchStart).Milliseconds()

		if resp.StatusCode != 200 {
			res.Error = fmt.Sprintf("[%s] HTTP %d", workerID, resp.StatusCode)
		}

		return res
	})
}
