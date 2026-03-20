package bench

import (
	"bufio"
	"context"
	"encoding/binary"
	"io"
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

	var events []Event
	r := bufio.NewReader(f)

	for {
		var tsBytes [8]byte
		if _, err := io.ReadFull(r, tsBytes[:]); err != nil {
			if err == io.EOF {
				break
			}
			return nil, err
		}

		var lenBytes [2]byte
		if _, err := io.ReadFull(r, lenBytes[:]); err != nil {
			return nil, err
		}

		keyLen := binary.LittleEndian.Uint16(lenBytes[:])
		keyBytes := make([]byte, keyLen)
		if _, err := io.ReadFull(r, keyBytes); err != nil {
			return nil, err
		}

		events = append(events, Event{Key: string(keyBytes)})
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
