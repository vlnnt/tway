package main

import (
	"context"
	"flag"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"tway/internal/app"
	"tway/internal/client"
	"tway/internal/client/kick"
	"tway/internal/client/twitch"
	"tway/internal/client/wtv"
	"tway/internal/client/youtube"
	"tway/internal/config"
	"tway/internal/notifier"
	"tway/internal/storage"
	"tway/internal/tray"
	"tway/internal/tui"

	"go.uber.org/zap"
)

func main() {
	exePath, err := os.Executable()
	if err != nil {
		return
	}

	exeDir := filepath.Dir(exePath)
	configPath := flag.String(
		"config",
		filepath.Join(exeDir, "tway.yaml"),
		"Path to config file",
	)

	iconPath := flag.String(
		"icon",
		filepath.Join(exeDir, "tway.ico"),
		"Path to icon file",
	)

	tuiMode := flag.Bool(
		"tui",
		false,
		"Run TUI",
	)

	flag.Parse()
	var logger *zap.Logger

	if *tuiMode {
		logger = zap.NewNop()
	} else {
		logger, err = zap.NewProduction()
		if err != nil {
			return
		}

		defer logger.Sync()
	}

	logger.Info(
		"Paths initialized",
		zap.String("Config", *configPath),
		zap.String("Icon", *iconPath),
	)

	logger.Info(
		"Loading config...",
		zap.String("Path", *configPath),
	)

	config, err := config.LoadConfig(*configPath)
	if err != nil {
		logger.Error(
			"config.LoadConfig",
			zap.Error(err),
		)
		return
	}

	checkInterval, err := time.ParseDuration(config.Check)
	if err != nil {
		logger.Error(
			"Parse check interval",
			zap.Error(err),
		)
		return
	}

	summaryInterval, err := time.ParseDuration(config.Summary.Interval)
	if err != nil {
		logger.Error(
			"Parse summary interval",
			zap.Error(err),
		)
		return
	}

	logger.Info(
		"Config has loaded",
		zap.Duration("Check interval", checkInterval),
		zap.Duration("Summary interval", summaryInterval),
		zap.Bool("Summary notify status", config.Summary.Enable),
		zap.Int("Twitch", len(config.Twitch.Channels)),
		zap.Int("Kick", len(config.Kick.Channels)),
		zap.Int("Youtube", len(config.Youtube.Channels)),
		zap.Int("WTV", len(config.WTV.Channels)),
	)

	logger.Info("Initializing clients...")
	platforms := []Platform{
		{
			Name:     "twitch",
			Channels: config.Twitch.Channels,
			Client: twitch.NewClient(
				logger,
				config.Twitch.Proxy.HTTP,
				config.Twitch.Proxy.Socks,
			),
		},
		{
			Name:     "kick",
			Channels: config.Kick.Channels,
			Client: kick.NewClient(
				logger,
				config.Kick.Proxy.HTTP,
				config.Kick.Proxy.Socks,
			),
		},
		{
			Name:     "youtube",
			Channels: config.Youtube.Channels,
			Client: youtube.NewClient(
				logger,
				config.Youtube.Proxy.HTTP,
				config.Youtube.Proxy.Socks,
			),
		},
		{
			Name:     "wtv",
			Channels: config.WTV.Channels,
			Client: wtv.NewClient(
				logger,
				config.WTV.Proxy.HTTP,
				config.WTV.Proxy.Socks,
			),
		},
	}

	logger.Info(
		"Clients initialized!",
		zap.Int(
			"Platforms",
			len(platforms),
		),
	)

	logger.Info("Initializing state storage...")
	stateStorage, err := storage.NewStateStorage(
		logger,
		filepath.Join(exeDir, "state.db"),
	)
	if err != nil {
		logger.Error(
			"Create storage",
			zap.Error(err),
		)
		return
	}
	defer stateStorage.Close()

	logger.Info("State storage initialized!")
	if *tuiMode {
		if err := tui.AttachConsole(); err != nil {
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
						state, err := stateStorage.Get(
							platform.Name,
							channel,
						)
						if err != nil {
							continue
						}

						if state == nil {
							continue
						}

						streams = append(
							streams,
							&client.Stream{
								ID:      state.StreamID,
								Channel: state.Channel,
								IsLive:  state.IsLive,
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
		); err != nil {
			return
		}
		return
	}

	logger.Info("Ensuring stream states...")
	for _, platform := range platforms {
		for _, channel := range platform.Channels {
			if err := stateStorage.Ensure(
				platform.Name,
				channel,
			); err != nil {
				logger.Error(
					"Failed to ensure stream state",
					zap.String("Platform", platform.Name),
					zap.String("Channel", channel),
					zap.Error(err),
				)
				return
			}
		}
	}

	logger.Info("Stream states ensured!")

	initializeStreamStates(
		logger,
		platforms,
		stateStorage,
	)

	logger.Info("Initializing notifier service...")
	notificationService, err := notifier.New(logger)
	if err != nil {
		logger.Error(
			"Create notifier",
			zap.Error(err),
		)
		return
	}
	defer notificationService.Close()

	logger.Info("Notifier service initialized!")

	logger.Info("Initializing applications...")
	applications := make(
		[]*app.App,
		0,
		len(platforms),
	)

	for _, platform := range platforms {
		application := app.NewApp(
			*iconPath,
			logger,
			platform.Name,
			platform.Channels,
			checkInterval,
			platform.Client,
			notificationService,
			stateStorage,
		)

		applications = append(
			applications,
			application,
		)
	}

	logger.Info(
		"Application initialized!",
		zap.Int(
			"Applications",
			len(applications),
		),
	)

	logger.Info("Creating notify context...")
	ctx, stop := signal.NotifyContext(
		context.Background(),
		os.Interrupt,
		syscall.SIGTERM,
	)
	defer stop()

	logger.Info("Notify context created!")

	logger.Info("Running applications...")
	for i, application := range applications {
		application := application
		platform := platforms[i]

		go func() {
			if err := application.Run(ctx); err != nil {
				logger.Error(
					"Application stopped",
					zap.String(
						"Platform",
						platform.Name,
					),
					zap.Error(err),
				)
				stop()
			}
		}()
	}

	if config.Summary.Enable {
		go runSummaryWorker(
			ctx,
			logger,
			summaryInterval,
			stateStorage,
			notificationService,
			*iconPath,
		)
	}

	logger.Info("Creating tray...")
	trayApp := tray.NewTray(
		logger,
		func() {
			logger.Info("Manual stream refresh requested")

			initializeStreamStates(
				logger,
				platforms,
				stateStorage,
			)

			if err := notificationService.Send(
				notifier.Notification{
					Title:   "tway",
					Message: "Streams status refreshed!",
					Icon:    *iconPath,
				},
			); err != nil {
				logger.Error(
					"Failed to send refresh notification",
					zap.Error(err),
				)
			}

			logger.Info("Manual stream refresh completed!")
		},
		func() {
			logger.Info("Tray exit event has requested!")
			stop()
		},
	)

	logger.Info("Tray created!")
	trayApp.Run()

	logger.Info("Tway stopped!")
}
