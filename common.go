package main

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
)

var ctx = context.Background()

type URLResult struct {
	URL       string    `json:"url"`
	Status    int       `json:"status"`
	Error     string    `json:"error,omitempty"`
	Duration  int64     `json:"duration_ms"`
	CheckedAt time.Time `json:"checked_at"`
	WorkerID  string    `json:"worker_id"`
}

type Stats struct {
	QueueLength int `json:"queue_length"`
	Success     int `json:"success"`
	Error       int `json:"error"`
	Processing  int `json:"processing"`
	Total       int `json:"total"`
}

func GetStats(rdb *redis.Client) Stats {
	queueLength, _ := rdb.LLen(ctx, "url_queue").Result()
	success, _ := rdb.Get(ctx, "success").Int()
	error, _ := rdb.Get(ctx, "error").Int()
	processing, _ := rdb.Get(ctx, "processing").Int()

	return Stats{
		QueueLength: int(queueLength),
		Success:     success,
		Error:       error,
		Processing:  processing,
		Total:       success + error + int(queueLength) + processing,
	}
}

func NewRedisClient(addr string) *redis.Client {
	return redis.NewClient(&redis.Options{
		Addr: addr,
	})
}
