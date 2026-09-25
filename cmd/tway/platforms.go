package main

import (
	"tway/internal/client"
	"tway/internal/client/kick"
	"tway/internal/client/twitch"
	"tway/internal/client/wtv"
	"tway/internal/client/youtube"
	"tway/internal/config"

	"go.uber.org/zap"
)

type Platform struct {
	Name     string
	Channels []string
	Client   client.Client
}

func platformsFromConfig(
	logger *zap.Logger,
	cfg *config.Config,
	initializeClients bool,
) []Platform {
	platforms := make([]Platform, 0, 4)
	if cfg.Twitch.Enable {
		platform := Platform{
			Name:     "twitch",
			Channels: cfg.Twitch.Channels,
		}

		if initializeClients {
			platform.Client = twitch.NewClient(
				logger,
				cfg.Twitch.Proxy.HTTP,
				cfg.Twitch.Proxy.Socks,
			)
		}

		platforms = append(
			platforms,
			platform,
		)
	}

	if cfg.Kick.Enable {
		platform := Platform{
			Name:     "kick",
			Channels: cfg.Kick.Channels,
		}

		if initializeClients {
			platform.Client = kick.NewClient(
				logger,
				cfg.Kick.Proxy.HTTP,
				cfg.Kick.Proxy.Socks,
			)
		}

		platforms = append(
			platforms,
			platform,
		)
	}

	if cfg.Youtube.Enable {
		platform := Platform{
			Name:     "youtube",
			Channels: cfg.Youtube.Channels,
		}

		if initializeClients {
			platform.Client = youtube.NewClient(
				logger,
				cfg.Youtube.Proxy.HTTP,
				cfg.Youtube.Proxy.Socks,
			)
		}

		platforms = append(
			platforms,
			platform,
		)
	}

	if cfg.WTV.Enable {
		platform := Platform{
			Name:     "wtv",
			Channels: cfg.WTV.Channels,
		}

		if initializeClients {
			platform.Client = wtv.NewClient(
				logger,
				cfg.WTV.Proxy.HTTP,
				cfg.WTV.Proxy.Socks,
			)
		}

		platforms = append(
			platforms,
			platform,
		)
	}

	return platforms
}

func streamURL(
	platform, channel string,
) string {
	switch platform {
	case "twitch":
		return "https://www.twitch.tv/" + channel

	case "kick":
		return "https://kick.com/" + channel

	case "youtube":
		return "https://www.youtube.com/@" + channel

	case "wtv":
		return "https://w.tv/" + channel

	default:
		return ""
	}
}
