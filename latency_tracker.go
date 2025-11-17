package main

import (
	"fmt"
	"sort"
	"sync"
	"time"
)

type LatencyTracker struct {
	mu        sync.Mutex
	latencies []int64
}

func NewLatencyTracker() *LatencyTracker {
	return &LatencyTracker{
		latencies: make([]int64, 0, 10000), //Preallocate for 10k entries
	}
}

func (lt *LatencyTracker) Record(latency time.Duration) {
	lt.mu.Lock()
	defer lt.mu.Unlock()
	lt.latencies = append(lt.latencies, latency.Microseconds())
}

func (lt *LatencyTracker) GetPercentiles() (p50, p95, p99, max int64) {
	lt.mu.Lock()
	defer lt.mu.Unlock()

	if len(lt.latencies) == 0 {
		return 0, 0, 0, 0
	}

	sorted := make([]int64, len(lt.latencies))
	copy(sorted, lt.latencies)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i] < sorted[j]
	})

	p50 = sorted[len(sorted)*50/100]
	p95 = sorted[len(sorted)*95/100]
	p99 = sorted[len(sorted)*99/100]
	max = sorted[len(sorted)-1]

	return p50, p95, p99, max
}

func (lt *LatencyTracker) PrintStats() {
	p50, p95, p99, max := lt.GetPercentiles()

	fmt.Printf("\n"+
		"════════════════════════════════════════\n"+
		"LATENCY PERCENTILES (%d samples)\n"+
		"════════════════════════════════════════\n"+
		"p50: %7.3f ms  (median)\n"+
		"p95: %7.3f ms  (95%% of requests)\n"+
		"p99: %7.3f ms  (99%% of requests)\n"+
		"max: %7.3f ms  (worst case)\n"+
		"════════════════════════════════════════\n",
		len(lt.latencies),
		float64(p50)/1000.0, // Convert µs to ms
		float64(p95)/1000.0,
		float64(p99)/1000.0,
		float64(max)/1000.0,
	)
}
