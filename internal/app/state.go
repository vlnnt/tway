package app

import "time"

type StreamState struct {
	Channel   string
	IsLive    bool
	StreamID  string
	UpdatedAt time.Time
}
