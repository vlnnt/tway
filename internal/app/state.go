package app

import "time"

type StreamState struct {
	IsLive    bool      `json:"is_live"`
	StreamID  string    `json:"stream_id"`
	UpdatedAt time.Time `json:"updated_at"`
}

type States map[string]StreamState
