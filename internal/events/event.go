package events

import "time"

type AccessEvent struct {
	Key        string
	OccurredAt time.Time
	ClientAddr string
}
