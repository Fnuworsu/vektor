package bench

import (
	"fmt"
	"sort"
	"time"
)

type LatencyStats struct {
	P50  time.Duration
	P95  time.Duration
	P99  time.Duration
	P999 time.Duration
	Max  time.Duration
}

type Histogram struct {
	latencies []time.Duration
}

func NewHistogram(capacity int) *Histogram {
	return &Histogram{
		latencies: make([]time.Duration, 0, capacity),
	}
}

func (h *Histogram) Record(d time.Duration) {
	h.latencies = append(h.latencies, d)
}

func (h *Histogram) Compute() LatencyStats {
	if len(h.latencies) == 0 {
		return LatencyStats{}
	}

	sort.Slice(h.latencies, func(i, j int) bool {
		return h.latencies[i] < h.latencies[j]
	})

	l := len(h.latencies)
	return LatencyStats{
		P50:  h.latencies[int(float64(l)*0.50)],
		P95:  h.latencies[int(float64(l)*0.95)],
		P99:  h.latencies[int(float64(l)*0.99)],
		P999: h.latencies[int(float64(l)*0.999)],
		Max:  h.latencies[l-1],
	}
}

func formatDuration(d time.Duration) string {
	if d < time.Millisecond {
		return fmt.Sprintf("%dµs", d.Microseconds())
	}
	return fmt.Sprintf("%.1fms", float64(d.Microseconds())/1000.0)
}

func PrintStats(mode string, stats LatencyStats, hitRate float64) {
	hrStr := "N/A"
	if hitRate >= 0 {
		hrStr = fmt.Sprintf("%.1f%%", hitRate*100)
	}

	fmt.Printf("%-10s %-8s %-8s %-8s %-8s %-8s %s\n",
		mode,
		formatDuration(stats.P50),
		formatDuration(stats.P95),
		formatDuration(stats.P99),
		formatDuration(stats.P999),
		formatDuration(stats.Max),
		hrStr,
	)
}

func PrintHeader() {
	fmt.Printf("%-10s %-8s %-8s %-8s %-8s %-8s %s\n",
		"Mode", "p50", "p95", "p99", "p999", "max", "prefetch_hit_rate")
}
