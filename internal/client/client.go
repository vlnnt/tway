package client

import "time"

type Stream struct {
	ID          string
	Channel     string
	Title       string
	Subcategory string
	StartedAt   time.Time
	IsLive      bool
	URL         string
}

type Client interface {
	GetStream(string) (*Stream, error)
}
