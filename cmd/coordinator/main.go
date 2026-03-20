package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Fnuworsu/vektor/internal/backend"
	"github.com/Fnuworsu/vektor/internal/cgobridge"
	"github.com/Fnuworsu/vektor/internal/coordinator"
	"github.com/Fnuworsu/vektor/internal/coordinator/policy"
	"github.com/Fnuworsu/vektor/internal/coordinator/tracker"
	"github.com/Fnuworsu/vektor/internal/events"
	"github.com/Fnuworsu/vektor/internal/proxy"
	"gopkg.in/yaml.v3"
)

type FullConfig struct {
	Backend backend.Config `yaml:"backend"`
	Proxy struct {
		ListenAddr string `yaml:"listen_addr"`
	} `yaml:"proxy"`
	Coordinator struct {
		PrefetchWorkers       int     `yaml:"prefetch_workers"`
		PrefetchThreshold     float64 `yaml:"prefetch_threshold"`
		MaxPrefetchQueueDepth int     `yaml:"max_prefetch_queue_depth"`
	} `yaml:"coordinator"`
	Engine struct {
		MarkovOrder    int `yaml:"markov_order"`
		MaxTrackedKeys int `yaml:"max_tracked_keys"`
	} `yaml:"engine"`
}

func main() {
	if err := run(); err != nil {
		log.Fatalf("Fatal: %v", err)
	}
}

func run() error {
	data, err := os.ReadFile("configs/vektor.yaml")
	if err != nil {
		return err
	}

	var cfg FullConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return err
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	store, err := backend.NewBackendStore(ctx, cfg.Backend)
	if err != nil {
		return err
	}
	defer store.Close()

	engine := cgobridge.NewEngine(cfg.Engine.MarkovOrder, cfg.Engine.MaxTrackedKeys, cfg.Coordinator.PrefetchThreshold)
	engine.Start()
	defer engine.Stop()

	t := tracker.NewTracker(2 * time.Second)
	p := policy.NewEngine(cfg.Coordinator.PrefetchThreshold)
	coord := coordinator.NewCoordinator(store, engine.Candidates(), t, p, cfg.Coordinator.PrefetchWorkers)
	coord.Start()
	defer coord.Stop()

	eventCh := make(chan events.AccessEvent, 100000)
	server := proxy.NewServer(cfg.Proxy.ListenAddr, store, eventCh)
	if err := server.Start(); err != nil {
		return err
	}

	go func() {
		for ev := range eventCh {
			t.CheckHit(ev.Key)
			_ = engine.PushEvent(ev.Key, ev.OccurredAt)
		}
	}()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	fmt.Printf("Vektor standalone coordinator/proxy listening on %s\n", cfg.Proxy.ListenAddr)
	<-sigCh

	fmt.Println("\nShutting down...")
	server.Stop()
	close(eventCh)

	return nil
}
