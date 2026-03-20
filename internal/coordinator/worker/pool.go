package worker

import (
	"context"
	"sync"
	"time"

	"github.com/Fnuworsu/vektor/internal/backend"
	"github.com/Fnuworsu/vektor/internal/cgobridge"
	"github.com/Fnuworsu/vektor/internal/coordinator/tracker"
)

type Pool struct {
	workers int
	store   backend.BackendStore
	tracker *tracker.Tracker
	wg      sync.WaitGroup
	ctx     context.Context
	cancel  context.CancelFunc
	ch      chan cgobridge.PrefetchCandidate
}

func NewPool(workers int, store backend.BackendStore, tracking *tracker.Tracker) *Pool {
	ctx, cancel := context.WithCancel(context.Background())
	return &Pool{
		workers: workers,
		store:   store,
		tracker: tracking,
		ctx:     ctx,
		cancel:  cancel,
		ch:      make(chan cgobridge.PrefetchCandidate),
	}
}

func (p *Pool) Start() {
	for i := 0; i < p.workers; i++ {
		p.wg.Add(1)
		go p.worker()
	}
}

func (p *Pool) Stop() {
	p.cancel()
	p.wg.Wait()
}

func (p *Pool) Submit(cand cgobridge.PrefetchCandidate) bool {
	select {
	case p.ch <- cand:
		return true
	default:
		return false
	}
}

func (p *Pool) worker() {
	defer p.wg.Done()
	for {
		select {
		case cand := <-p.ch:
			p.process(cand)
		case <-p.ctx.Done():
			return
		}
	}
}

func (p *Pool) process(cand cgobridge.PrefetchCandidate) {
	ctx, cancel := context.WithTimeout(p.ctx, 100*time.Millisecond)
	defer cancel()

	val, err := p.store.Get(ctx, cand.Key)
	if err != nil {
		p.tracker.RecordMiss()
		return
	}

	if val != nil {
		p.tracker.RecordMiss()
		return
	}

	time.Sleep(1 * time.Microsecond)
	p.tracker.RecordIssued(cand.Key)
}
