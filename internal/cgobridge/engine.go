package cgobridge

import "time"

type PrefetchCandidate struct {
	Key         string
	Probability float64
}

type Engine interface {
	Start()
	Stop()
	PushEvent(key string, ts time.Time) error
	Candidates() <-chan PrefetchCandidate
}
