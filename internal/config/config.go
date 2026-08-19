package config

import (
	"os"

	"gopkg.in/yaml.v2"
)

type Config struct {
	Check   string  `yaml:"check"`
	Summary Summary `yaml:"summary"`
	Twitch  Twitch  `yaml:"twitch"`
	Kick    Kick    `yaml:"kick"`
	Youtube Youtube `yaml:"youtube"`
	WTV     WTV     `yaml:"wtv"`
}

type Summary struct {
	Enable   bool   `yaml:"enable"`
	Interval string `yaml:"interval"`
}

type Proxy struct {
	HTTP  string `yaml:"http"`
	Socks string `yaml:"socks"`
}

type Platform struct {
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
	err = yaml.Unmarshal(config, settings)
	if err != nil {
		return nil, err
	}

	return settings, nil
}

func Get() *Config {
	c := &Config{}
	return c
}
