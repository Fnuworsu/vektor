package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/Fnuworsu/vektor/internal/backend"
	"github.com/Fnuworsu/vektor/internal/cgobridge"
	"github.com/Fnuworsu/vektor/internal/events"
	"github.com/Fnuworsu/vektor/internal/proxy"
	"gopkg.in/yaml.v3"
)

type ProxyConfig struct {
	ListenAddr string `yaml:"listen_addr"`
}

type EngineConfig struct {
	MarkovOrder    int `yaml:"markov_order"`
	MaxTrackedKeys int `yaml:"max_tracked_keys"`
}

type CoordinatorConfig struct {
	PrefetchThreshold float64 `yaml:"prefetch_threshold"`
}

type FullConfig struct {
	Backend     backend.Config    `yaml:"backend"`
	Proxy       ProxyConfig       `yaml:"proxy"`
	Engine      EngineConfig      `yaml:"engine"`
	Coordinator CoordinatorConfig `yaml:"coordinator"`
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

	eventCh := make(chan events.AccessEvent, 100000)

	server := proxy.NewServer(cfg.Proxy.ListenAddr, store, eventCh)
	if err := server.Start(); err != nil {
		return err
	}

	go func() {
		for ev := range eventCh {
			_ = engine.PushEvent(ev.Key, ev.OccurredAt)
		}
	}()

	go func() {
		for range engine.Candidates() {
		}
	}()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	fmt.Printf("Vektor proxy listening on %s\n", cfg.Proxy.ListenAddr)

	<-sigCh

	fmt.Println("\nShutting down proxy...")
	server.Stop()
	close(eventCh)

	return nil
}
