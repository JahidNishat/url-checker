package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
)

func main() {
	//Load Config
	config := LoadConfig()

	rdb := NewRedisClient(config.RedisAddr)

	http.HandleFunc("/stats", func(w http.ResponseWriter, r *http.Request) {
		stats := GetStats(rdb)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(stats)
	})

	http.HandleFunc("/results", func(w http.ResponseWriter, r *http.Request) {
		results, _ := rdb.LRange(ctx, "results", 0, 99).Result()

		var urlResults []URLResult
		for _, result := range results {
			var res URLResult
			json.Unmarshal([]byte(result), &res)
			urlResults = append(urlResults, res)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(urlResults)
	})

	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		if err := rdb.Ping(ctx).Err(); err != nil {
			w.WriteHeader(http.StatusServiceUnavailable)
			fmt.Fprintf(w, "Redis is unavailable: %v", err)
			return
		}
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, "OK")
	})

	log.Println("API Server starting on :8080")
	log.Println("Endpoints:")
	log.Println("  GET /stats   - Current statistics")
	log.Println("  GET /results - Last 100 results")
	log.Println("  GET /health  - Health check")

	http.ListenAndServe(":8080", nil)
}
