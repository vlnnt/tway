package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"

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
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	settings := Default()
	if err := yaml.Unmarshal(
		data,
		settings,
	); err != nil {
		return nil, err
	}

	return settings, nil
}

func SaveConfig(
	path string,
	config *Config,
) error {
	var builder strings.Builder

	fmt.Fprintf(
		&builder,
		"check: %s\n\n",
		strconv.Quote(config.Check),
	)

	fmt.Fprintf(
		&builder,
		"summary:\n"+
			"  enable: %t\n"+
			"  interval: %s\n\n",
		config.Summary.Enable,
		strconv.Quote(config.Summary.Interval),
	)

	fmt.Fprintf(
		&builder,
		"ui:\n"+
			"  show_streamers: %t\n\n",
		config.UI.ShowStreamers,
	)

	writePlatform(
		&builder,
		"twitch",
		&config.Twitch.Platform,
	)

	writePlatform(
		&builder,
		"kick",
		&config.Kick.Platform,
	)

	writePlatform(
		&builder,
		"youtube",
		&config.Youtube.Platform,
	)

	writePlatform(
		&builder,
		"wtv",
		&config.WTV.Platform,
	)

	return os.WriteFile(
		path,
		[]byte(builder.String()),
		0644,
	)
}

func writePlatform(
	builder *strings.Builder,
	name string,
	platform *Platform,
) {
	fmt.Fprintf(
		builder,
		"%s:\n"+
			"  enable: %t\n"+
			"  proxy:\n"+
			"    http: %s\n"+
			"    socks: %s\n",
		name,
		platform.Enable,
		strconv.Quote(platform.Proxy.HTTP),
		strconv.Quote(platform.Proxy.Socks),
	)

	if len(platform.Channels) == 0 {
		builder.WriteString(
			"  channels: []\n\n",
		)

		return
	}

	builder.WriteString(
		"  channels:\n",
	)

	for _, channel := range platform.Channels {
		fmt.Fprintf(
			builder,
			"    - %s\n",
			strconv.Quote(channel),
		)
	}

	builder.WriteString("\n")
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
