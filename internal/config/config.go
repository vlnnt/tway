package config

import (
	"encoding/json"
	"fmt"
	"time"
)

type Config struct {
	CheckInterval   Duration `json:"check_interval"`
	SummaryInterval Duration `json:"summary_interval"`
	Twitch          Twitch   `json:"twitch"`
	Kick            Kick     `json:"kick"`
	Youtube         Youtube  `json:"youtube"`
}

type Platform struct {
	HTTPProxy  string   `json:"http_proxy"`
	SocksProxy string   `json:"socks_proxy"`
	Channels   []string `json:"channels"`
}

type Twitch struct {
	Platform
}

type Kick struct {
	Platform
}

type Youtube struct {
	Platform
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
