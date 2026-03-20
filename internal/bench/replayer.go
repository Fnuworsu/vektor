package bench

import (
	"context"
	"encoding/csv"
	"os"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
)

type Event struct {
	Key string
}

type Replayer struct {
	addr    string
	workers int
	rate    int
	events  []Event
	hist    *Histogram
}

func NewReplayer(addr string, workers int, rate int, tracePath string) (*Replayer, error) {
	f, err := os.Open(tracePath)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	r := csv.NewReader(f)
	records, err := r.ReadAll()
	if err != nil {
		return nil, err
	}

	events := make([]Event, 0, len(records))
	for i, rec := range records {
		if i == 0 && rec[1] == "key" {
			continue
		}
		events = append(events, Event{Key: rec[1]})
	}

	return &Replayer{
		addr:    addr,
		workers: workers,
		rate:    rate,
		events:  events,
		hist:    NewHistogram(len(events)),
	}, nil
}

func (r *Replayer) Run(ctx context.Context, duration time.Duration) LatencyStats {
	client := redis.NewClient(&redis.Options{Addr: r.addr, PoolSize: r.workers})
	defer client.Close()

	ch := make(chan Event, r.workers*2)
	var wg sync.WaitGroup
	var mu sync.Mutex

	for i := 0; i < r.workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for ev := range ch {
				start := time.Now()
				_ = client.Get(ctx, ev.Key).Err()
				d := time.Since(start)
				mu.Lock()
				r.hist.Record(d)
				mu.Unlock()
			}
		}()
	}

	ticker := time.NewTicker(time.Second / time.Duration(r.rate))
	defer ticker.Stop()

	timer := time.NewTimer(duration)
	defer timer.Stop()

	count := 0
	total := len(r.events)

	for {
		select {
		case <-ctx.Done():
			close(ch)
			wg.Wait()
			return r.hist.Compute()
		case <-timer.C:
			close(ch)
			wg.Wait()
			return r.hist.Compute()
		case <-ticker.C:
			if count >= total {
				close(ch)
				wg.Wait()
				return r.hist.Compute()
			}
			ch <- r.events[count]
			count++
		}
	}
}
