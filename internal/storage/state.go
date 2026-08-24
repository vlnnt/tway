package storage

import "time"

type StreamState struct {
	Platform     string
	Channel      string
	IsLive       bool
	LastStreamAt time.Time
	StartedAt    time.Time
}
