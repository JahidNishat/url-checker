package main

import (
	"context"
	"encoding/json"
	"fmt"
	"sync/atomic"
	"time"

	lru "github.com/hashicorp/golang-lru/v2"
	"github.com/redis/go-redis/v9"
)

type cacheEntry struct {
	result    URLResult
	timestamp time.Time
}

type CacheManager struct {
	l1       *lru.Cache[string, cacheEntry]
	l2       *redis.Client
	stampede *StampedePreventer

	//Metrics (use atomic for concurrency safety)
	l1Hits int64
	l2Hits int64
	origin int64
}

func NewCacheManager(l1Size int, redisClient *redis.Client, stampede *StampedePreventer) (*CacheManager, error) {
	l1Cache, err := lru.New[string, cacheEntry](l1Size)
	if err != nil {
		return nil, err
	}
	return &CacheManager{l1: l1Cache, l2: redisClient, stampede: stampede}, nil
}

func (cm *CacheManager) Get(ctx context.Context, url string, fetchFunc func(string) URLResult) URLResult {
	//Check L1 cache (in-memory)
	entry, ok := cm.l1.Get(url)
	if ok {
		if time.Since(entry.timestamp) < 60*time.Second {
			atomic.AddInt64(&cm.l1Hits, 1)
			return entry.result
		} else {
			cm.l1.Remove(url)
		}
	}

	//Check L2 cache (Redis)
	cacheKey := fmt.Sprintf("cache:%s", url)
	cache, err := cm.l2.Get(ctx, cacheKey).Bytes()
	if err == nil {
		atomic.AddInt64(&cm.l2Hits, 1)
		cacheRes := URLResult{}
		json.Unmarshal(cache, &cacheRes)

		if cacheRes.Status == 200 {
			cm.l1.Add(url, cacheEntry{cacheRes, time.Now()})
		}

		return cacheRes
	}

	//Fetch URL
	result := stampede.Fetch(url, fetchFunc)
	atomic.AddInt64(&cm.origin, 1)
	if result.Status == 200 {
		cm.l1.Add(url, cacheEntry{result, time.Now()})
	}
	resultByte, _ := json.Marshal(result)
	cm.l2.Set(ctx, cacheKey, resultByte, 5*time.Minute)
	return result
}

func (cm *CacheManager) GetStats() (l1Hits, l2Hits, originFetch int64) {
	return atomic.LoadInt64(&cm.l1Hits), atomic.LoadInt64(&cm.l2Hits), atomic.LoadInt64(&cm.origin)
}
