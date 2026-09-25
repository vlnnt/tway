package config

import (
	"os"

	"gopkg.in/yaml.v2"
)

type Config struct {
	Check   string  `yaml:"check"`
	Summary Summary `yaml:"summary"`
	UI      UI      `yaml:"ui"`
	Twitch  Twitch  `yaml:"twitch"`
	Kick    Kick    `yaml:"kick"`
	Youtube Youtube `yaml:"youtube"`
	WTV     WTV     `yaml:"wtv"`
}

type Summary struct {
	Enable   bool   `yaml:"enable"`
	Interval string `yaml:"interval"`
}

type UI struct {
	ShowStreamers bool `yaml:"show_streamers"`
}

type Proxy struct {
	HTTP  string `yaml:"http"`
	Socks string `yaml:"socks"`
}

type Platform struct {
	Enable   bool     `yaml:"enable"`
	Proxy    Proxy    `yaml:"proxy"`
	Channels []string `yaml:"channels"`
}

type Twitch struct {
	Platform `yaml:",inline"`
}

type Kick struct {
	Platform `yaml:",inline"`
}

type Youtube struct {
	Platform `yaml:",inline"`
}

type WTV struct {
	Platform `yaml:",inline"`
}

func LoadConfig(
	path string,
) (*Config, error) {
	config, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	settings := &Config{}
	if err := yaml.Unmarshal(config, settings); err != nil {
		return nil, err
	}

	return settings, nil
}

func SaveConfig(
	path string, config *Config,
) error {
	data, err := yaml.Marshal(config)
	if err != nil {
		return err
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return err
	}

	return nil
}

func Default() *Config {
	return &Config{
		Check: "2m",
		Summary: Summary{
			Enable:   false,
			Interval: "10m",
		},
		UI: UI{
			ShowStreamers: true,
		},
	}
}
