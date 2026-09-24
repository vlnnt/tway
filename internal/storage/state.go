package storage

import "time"

type StreamKey struct {
	Platform string
	Channel  string
}

type StreamState struct {
	Platform     string
	Channel      string
	IsTracked    bool
	IsLive       bool
	LastStreamAt time.Time
	StartedAt    time.Time
}
