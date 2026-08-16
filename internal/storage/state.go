package storage

import "time"

type StreamState struct {
	Platform  string
	Channel   string
	IsLive    bool
	StreamID  string
	UpdatedAt time.Time
}
