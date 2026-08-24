package client

import "time"

type Stream struct {
	Channel      string
	Title        string
	Subcategory  string
	LastStreamAt time.Time
	IsLive       bool
	URL          string
}

type Client interface {
	GetStream(string) (*Stream, error)
}
