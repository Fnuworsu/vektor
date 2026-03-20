package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/Fnuworsu/vektor/internal/bench"
)

type OutputResult struct {
	Mode            string  `json:"mode"`
	P50             string  `json:"p50"`
	P95             string  `json:"p95"`
	P99             string  `json:"p99"`
	P999            string  `json:"p999"`
	Max             string  `json:"max"`
	PrefetchHitRate float64 `json:"prefetch_hit_rate"`
}

func main() {
	mode := flag.String("mode", "baseline", "baseline or vektor")
	trace := flag.String("trace", "", "path to trace CSV")
	workers := flag.Int("workers", 10, "concurrent workers")
	duration := flag.String("duration", "10s", "how long to run")
	rate := flag.Int("rate", 10000, "events per second")
	flag.Parse()

	if *trace == "" {
		log.Fatal("trace path required")
	}

	dur, err := time.ParseDuration(*duration)
	if err != nil {
		log.Fatal(err)
	}

	addr := "localhost:6379"
	if *mode == "vektor" {
		addr = "localhost:6380"
	}

	r, err := bench.NewReplayer(addr, *workers, *rate, *trace)
	if err != nil {
		log.Fatal(err)
	}

	ctx := context.Background()
	stats := r.Run(ctx, dur)

	hitRate := -1.0

	bench.PrintHeader()
	bench.PrintStats(*mode, stats, hitRate)

	os.MkdirAll("benchmarks/results", 0755)
	fname := filepath.Join("benchmarks", "results", fmt.Sprintf("%d.json", time.Now().Unix()))
	res := OutputResult{
		Mode:            *mode,
		P50:             fmt.Sprintf("%dµs", stats.P50.Microseconds()),
		P95:             fmt.Sprintf("%dµs", stats.P95.Microseconds()),
		P99:             fmt.Sprintf("%dµs", stats.P99.Microseconds()),
		P999:            fmt.Sprintf("%dµs", stats.P999.Microseconds()),
		Max:             fmt.Sprintf("%dµs", stats.Max.Microseconds()),
		PrefetchHitRate: hitRate,
	}
	b, _ := json.MarshalIndent(res, "", "  ")
	os.WriteFile(fname, b, 0644)
}
