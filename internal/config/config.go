package config

import (
	"encoding/json"
	"fmt"
	"time"
)

type Config struct {
	CheckInterval Duration   `json:"check_interval"`
	Streamers     []Streamer `json:"streamers"`
}

type Streamer struct {
	Channel string `json:"channel"`
	Enabled bool   `json:"enabled"`
}

type Duration struct {
	time.Duration
}

func (d *Duration) UnmarshalJSON(
	data []byte,
) error {
	var value string
	if err := json.Unmarshal(data, &value); err != nil {
		return fmt.Errorf("decode duration: %w", err)
	}

	duration, err := time.ParseDuration(value)
	if err != nil {
		return fmt.Errorf("parse duration %q: %w", value, err)
	}

	d.Duration = duration
	return nil
}
