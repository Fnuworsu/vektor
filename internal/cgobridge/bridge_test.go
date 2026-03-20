package cgobridge

import (
	"testing"
	"time"
)

func TestEngineIntegration(t *testing.T) {
	engine := NewEngine(1, 100, 0.6)
	engine.Start()
	defer engine.Stop()

	for i := 0; i < 500; i++ {
		err := engine.PushEvent("A", time.Now())
		if err != nil {
			t.Fatalf("push failed: %v", err)
		}
		engine.PushEvent("B", time.Now())
	}

	timeout := time.After(2 * time.Second)
	foundB := false

loop:
	for {
		select {
		case cand := <-engine.Candidates():
			if cand.Key == "B" && cand.Probability >= 0.6 {
				foundB = true
				break loop
			}
		case <-timeout:
			break loop
		}
	}

	if !foundB {
		t.Fatal("expected prefetch candidate B with prob >= 0.6")
	}
}
