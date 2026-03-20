package tracker

import (
	"sync"
	"time"
)

type Snapshot struct {
	PrefetchIssued   uint64
	PrefetchHit      uint64
	PrefetchMiss     uint64
	PrefetchDropped  uint64
	TotalGetsProxied uint64
}

type Tracker struct {
	issued   uint64
	hit      uint64
	miss     uint64
	dropped  uint64
	proxied  uint64

	mu sync.RWMutex

	recent map[string]time.Time
	ttl    time.Duration
}

func NewTracker(ttl time.Duration) *Tracker {
	return &Tracker{
		recent: make(map[string]time.Time),
		ttl:    ttl,
	}
}

func (t *Tracker) RecordIssued(key string) {
	t.mu.Lock()
	t.issued++
	t.recent[key] = time.Now().Add(t.ttl)
	t.mu.Unlock()
}

func (t *Tracker) RecordDropped() {
	t.mu.Lock()
	t.dropped++
	t.mu.Unlock()
}

func (t *Tracker) RecordMiss() {
	t.mu.Lock()
	t.miss++
	t.mu.Unlock()
}

func (t *Tracker) CheckHit(key string) bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.proxied++

	expiry, exists := t.recent[key]
	if !exists {
		return false
	}
	
	if time.Now().After(expiry) {
		delete(t.recent, key)
		return false
	}
	
	t.hit++
	delete(t.recent, key)
	return true
}

func (t *Tracker) Snapshot() Snapshot {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return Snapshot{
		PrefetchIssued:   t.issued,
		PrefetchHit:      t.hit,
		PrefetchMiss:     t.miss,
		PrefetchDropped:  t.dropped,
		TotalGetsProxied: t.proxied,
	}
}
