package main

import (
	"context"
	"log"
	"os"

	"github.com/Fnuworsu/vektor/internal/backend"
	"gopkg.in/yaml.v3"
)

type FullConfig struct {
	Backend backend.Config `yaml:"backend"`
}

func main() {
	if err := run(); err != nil {
		log.Fatalf("Fatal error: %v", err)
	}
}

func run() error {
	ctx := context.Background()

	data, err := os.ReadFile("configs/vektor.yaml")
	if err != nil {
		return err
	}

	var cfg FullConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return err
	}

	store, err := backend.NewBackendStore(ctx, cfg.Backend)
	if err != nil {
		return err
	}
	defer store.Close()

	key := "vektor:smoke"
	val := []byte("smoke_test_passed")

	log.Printf("Setting key %s to %s", key, val)
	if err := store.Set(ctx, key, val); err != nil {
		return err
	}

	log.Printf("Getting key %s", key)
	fetched, err := store.Get(ctx, key)
	if err != nil {
		return err
	}

	if string(fetched) == string(val) {
		log.Println("Smoke test completed successfully. Data matched.")
	} else {
		log.Printf("Smoke test failed. Data mismatch. Expected %s, got %s", string(val), string(fetched))
	}

	return nil
}
