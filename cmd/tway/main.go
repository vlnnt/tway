package main

import (
	"context"
	"encoding/json"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"tway/internal/app"
	"tway/internal/config"
	"tway/internal/notifier"
	"tway/internal/tray"
	"tway/internal/twitch"

	"go.uber.org/zap"
)

func main() {
	logger, _ := zap.NewProduction()
	defer logger.Sync()

	exePath, err := os.Executable()
	if err != nil {
		logger.Error("main.os.Executable", zap.Error(err))
	}

	configPath := filepath.Join(
		filepath.Dir(exePath),
		"stream.json",
	)

	iconPath := filepath.Join(
		filepath.Dir(exePath),
		"twitch.ico",
	)

	logger.Info("Icon path",
		zap.String("Path", iconPath),
	)

	if len(os.Args) >= 2 {
		configPath = os.Args[1]
	}

	logger.Info("Loading config ...", zap.String("Path", configPath))
	file, err := os.Open(configPath)
	if err != nil {
		logger.Error("main.os.Open", zap.Error(err))
	}
	defer file.Close()

	var config config.Config
	if err := json.NewDecoder(file).Decode(&config); err != nil {
		logger.Error("main.json.NewDecoder.Decode", zap.Error(err))
	}

	logger.Info("Config has loaded",
		zap.Duration("Interval", config.CheckInterval.Duration),
		zap.Int("Streamers", len(config.Streamers)),
	)

	logger.Info("Initializing twitch client ...")
	twitchClient := twitch.NewClient(logger)

	logger.Info("Twitch client initialized!")

	logger.Info("Initializing notifier service ...")
	notificationService, err := notifier.New(logger)
	if err != nil {
		logger.Error("Create notifier", zap.Error(err))
	}
	defer notificationService.Close()

	logger.Info("Notifier service initialized!")

	logger.Info("Initializing state storage ...")
	storage, err := app.NewStateStorage("state.db")
	if err != nil {
		logger.Error("Create storage", zap.Error(err))
	}

	logger.Info("State storage initialized!")

	logger.Info("Initializing application ...")
	application := app.New(
		iconPath,
		logger,
		&config,
		twitchClient,
		notificationService,
		storage,
	)

	logger.Info("Application initialized!")

	logger.Info("Creating notify context ...")
	ctx, stop := signal.NotifyContext(
		context.Background(),
		os.Interrupt,
		syscall.SIGTERM,
	)
	defer stop()

	logger.Info("Notify context created!")

	logger.Info("Running application ...")
	go func() {
		if err := application.Run(ctx); err != nil {
			logger.Error("Application stopped", zap.Error(err))
			stop()
		}
	}()

	logger.Info("Application started!")

	logger.Info("Creating tray ...")
	trayApp := tray.New(
		iconPath,
		logger,
		&config,
		twitchClient,
		notificationService,
		func() {
			logger.Info("Tray exit event has requested!")
			stop()
		},
	)

	logger.Info("Tray created!")

	trayApp.Run()
	logger.Info("Tway stopped!")
}
