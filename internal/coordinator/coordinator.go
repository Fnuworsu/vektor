package coordinator

import (
	"context"

	"github.com/Fnuworsu/vektor/internal/backend"
	"github.com/Fnuworsu/vektor/internal/cgobridge"
	"github.com/Fnuworsu/vektor/internal/coordinator/policy"
	"github.com/Fnuworsu/vektor/internal/coordinator/tracker"
	"github.com/Fnuworsu/vektor/internal/coordinator/worker"
)

type Coordinator struct {
	tracker  *tracker.Tracker
	policy   *policy.Engine
	pool     *worker.Pool
	candCh   <-chan cgobridge.PrefetchCandidate
	ctx      context.Context
	cancel   context.CancelFunc
	done     chan struct{}
}

func NewCoordinator(store backend.BackendStore, candCh <-chan cgobridge.PrefetchCandidate, tracking *tracker.Tracker, p *policy.Engine, workers int) *Coordinator {
	ctx, cancel := context.WithCancel(context.Background())
	return &Coordinator{
		tracker: tracking,
		policy:  p,
		pool:    worker.NewPool(workers, store, tracking),
		candCh:  candCh,
		ctx:     ctx,
		cancel:  cancel,
		done:    make(chan struct{}),
	}
}

func (c *Coordinator) Start() {
	c.pool.Start()
	go c.loop()
}

func (c *Coordinator) Stop() {
	c.cancel()
	<-c.done
	c.pool.Stop()
}

func (c *Coordinator) loop() {
	defer close(c.done)
	for {
		select {
		case cand := <-c.candCh:
			if c.policy.ShouldPrefetch(cand.Probability) {
				if !c.pool.Submit(cand) {
					c.tracker.RecordDropped()
				}
			}
		case <-c.ctx.Done():
			return
		}
	}
}
