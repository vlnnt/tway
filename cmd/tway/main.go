package main

import (
	"context"
	"encoding/json"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"tway/internal/app"
	"tway/internal/client"
	"tway/internal/client/kick"
	"tway/internal/client/twitch"
	"tway/internal/client/youtube"
	"tway/internal/config"
	"tway/internal/notifier"
	"tway/internal/storage"
	"tway/internal/tray"
	"tway/internal/tui"

	"go.uber.org/zap"
)

func main() {
	logger, err := zap.NewProduction()
	if err != nil {
		logger.Error("main.zap.NewProduction", zap.Error(err))
		return
	}
	defer logger.Sync()

	exePath, err := os.Executable()
	if err != nil {
		logger.Error("main.os.Executable", zap.Error(err))
		return
	}

	exeDir := filepath.Dir(exePath)
	configPath := filepath.Join(exeDir, "config.json")
	iconPath := filepath.Join(exeDir, "tway.ico")

	logger.Info("Icon path", zap.String("Path", iconPath))
	if len(os.Args) >= 2 && os.Args[1] != "--tui" {
		configPath = os.Args[1]
	}

	logger.Info("Loading config ...", zap.String("Path", configPath))
	file, err := os.Open(configPath)
	if err != nil {
		logger.Error("main.os.Open", zap.Error(err))
		return
	}
	defer file.Close()

	var config config.Config
	if err := json.NewDecoder(file).Decode(&config); err != nil {
		logger.Error("main.json.NewDecoder.Decode", zap.Error(err))
		return
	}

	logger.Info(
		"Config has loaded",
		zap.Duration("Check interval", config.CheckInterval.Duration),
		zap.Duration("Summary interval", config.SummaryInterval.Duration),
		zap.Int("Twitch", len(config.Twitch.Channels)),
		zap.Int("Kick", len(config.Kick.Channels)),
		zap.Int("Youtube", len(config.Youtube.Channels)),
	)

	logger.Info("Initializing clients ...")
	platforms := []Platform{
		{
			Name:     "twitch",
			Channels: config.Twitch.Channels,
			Client: twitch.NewClient(
				logger,
				config.Twitch.HTTPProxy,
				config.Twitch.SocksProxy,
			),
		},
		{
			Name:     "kick",
			Channels: config.Kick.Channels,
			Client: kick.NewClient(
				logger,
				config.Kick.HTTPProxy,
				config.Kick.SocksProxy,
			),
		},
		{
			Name:     "youtube",
			Channels: config.Youtube.Channels,
			Client: youtube.NewClient(
				logger,
				config.Youtube.HTTPProxy,
				config.Youtube.SocksProxy,
			),
		},
	}

	logger.Info(
		"Clients initialized!",
		zap.Int("Platforms", len(platforms)),
	)

	if len(os.Args) >= 2 && os.Args[1] == "--tui" {
		logger.Info("Starting TUI ...")
		if err := tui.AttachConsole(); err != nil {
			logger.Error(
				"main.AttachConsole",
				zap.Error(err),
			)
			return
		}

		var streams []*client.Stream
		for _, platform := range platforms {
			for _, channel := range platform.Channels {
				stream, err := platform.Client.GetStream(channel)
				if err != nil {
					logger.Error(
						"main.GetStream",
						zap.String("Platform", platform.Name),
						zap.String("Channel", channel),
						zap.Error(err),
					)
					continue
				}
				streams = append(streams, stream)
			}
		}

		ui := tui.NewTUI()
		if err := ui.ShowStreamers(streams); err != nil {
			logger.Error(
				"TUI stopped with error",
				zap.Error(err),
			)
		}
		return
	}

	logger.Info("Initializing notifier service ...")
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

	logger.Info("Initializing state storage ...")
	storage, err := storage.NewStateStorage(
		filepath.Join(exeDir, "state.db"),
	)
	if err != nil {
		logger.Error("Create storage", zap.Error(err))
		return
	}

	logger.Info("State storage initialized!")

	logger.Info("Initializing applications ...")
	applications := make(
		[]*app.App,
		0,
		len(platforms),
	)

	for _, platform := range platforms {
		application := app.NewApp(
			iconPath,
			logger,
			platform.Name,
			platform.Channels,
			config.CheckInterval.Duration,
			config.SummaryInterval.Duration,
			platform.Client,
			notificationService,
			storage,
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

	logger.Info("Creating notify context ...")
	ctx, stop := signal.NotifyContext(
		context.Background(),
		os.Interrupt,
		syscall.SIGTERM,
	)

	defer stop()
	logger.Info("Notify context created!")

	logger.Info("Running applications ...")

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

	go runSummaryWorker(
		ctx,
		logger,
		platforms,
		config.SummaryInterval.Duration,
		notificationService,
		iconPath,
	)

	logger.Info("Creating tray ...")
	trayApp := tray.NewTray(
		logger,
		func() {
			logger.Info("Tray exit event has requested!")
			stop()
		},
	)

	logger.Info("Tray created!")
	trayApp.Run()

	logger.Info("Tway stopped!")
}
