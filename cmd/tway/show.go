package main

import (
	"errors"
	"tway/internal/client"
	"tway/internal/config"
	"tway/internal/i18n"
	"tway/internal/storage"
	"tway/internal/tui"

	"go.uber.org/zap"
)

func runStreamersTUI(
	cfg *config.Config,
	logger *zap.Logger,
	stateStorage *storage.StateStorage,
) {
	texts := i18n.Get(cfg.Language)
	platforms := platformsFromConfig(
		logger,
		cfg,
		false,
	)

	platformNames := make(
		[]string,
		0,
		len(platforms),
	)

	for _, platform := range platforms {
		switch platform.Name {
		case "twitch":
			platformNames = append(
				platformNames,
				"Twitch",
			)

		case "kick":
			platformNames = append(
				platformNames,
				"Kick",
			)

		case "youtube":
			platformNames = append(
				platformNames,
				"YouTube",
			)

		case "wtv":
			platformNames = append(
				platformNames,
				"W.TV",
			)
		}
	}

	if err := tui.AttachConsole(); err != nil {
		if errors.Is(err, tui.ErrNoConsole) {
			if err := tui.OpenTerminal("--tui"); err != nil {
				logger.Error(
					"Open TUI terminal",
					zap.Error(err),
				)
			}
			return
		}

		logger.Error(
			"main.AttachConsole",
			zap.Error(err),
		)
		return
	}

	ui := tui.NewTUI()
	if err := ui.ShowStreamers(
		func() ([]*client.Stream, error) {
			var streams []*client.Stream
			for _, platform := range platforms {
				for _, channel := range platform.Channels {
					state, err := stateStorage.Get(platform.Name, channel)
					if err != nil {
						continue
					}

					if state == nil {
						continue
					}

					streams = append(
						streams,
						&client.Stream{
							Channel:      state.Channel,
							IsLive:       state.IsLive,
							LastStreamAt: state.LastStreamAt,
							StartedAt:    state.StartedAt,
							URL: streamURL(
								state.Platform,
								state.Channel,
							),
						},
					)
				}
			}
			return streams, nil
		},
		platformNames,
		texts,
	); err != nil {
		return
	}
}
