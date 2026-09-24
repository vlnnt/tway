package main

import (
	"context"
	"flag"
	"os"
	"os/signal"
	"path/filepath"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"tway/internal/client"
	"tway/internal/config"
	"tway/internal/notifier"
	"tway/internal/storage"
	"tway/internal/tray"
	"tway/internal/tui"

	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
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

	setupMode := flag.Bool(
		"setup",
		false,
		"Open configuration setup",
	)

	flag.Parse()
	var logger *zap.Logger

	if *tuiMode || *setupMode {
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

	cfg, err := config.LoadConfig(*configPath)
	firstRun := false

	if err != nil {
		if !os.IsNotExist(err) {
			logger.Error(
				"config.LoadConfig",
				zap.Error(err),
			)

			return
		}

		cfg = config.Default()
		firstRun = true
	}

	if firstRun || *setupMode {
		if err := tui.AttachConsole(); err != nil {
			logger.Error(
				"tui.AttachConsole",
				zap.Error(err),
			)
			return
		}

		ui := tui.NewTUI()
		saved, err := ui.ShowSetup(cfg)
		if err != nil {
			logger.Error(
				"Show setup",
				zap.Error(err),
			)
			return
		}

		if !saved {
			return
		}

		if err := config.SaveConfig(*configPath, cfg); err != nil {
			logger.Error(
				"config.SaveConfig",
				zap.Error(err),
			)
			return
		}

		if *setupMode {
			return
		}
	}

	logger.Info(
		"Config has loaded",
		zap.String("Check interval", cfg.Check),
		zap.String("Summary interval", cfg.Summary.Interval),
		zap.Bool("Summary notify status", cfg.Summary.Enable),
		zap.Int("Twitch", len(cfg.Twitch.Channels)),
		zap.Int("Kick", len(cfg.Kick.Channels)),
		zap.Int("Youtube", len(cfg.Youtube.Channels)),
		zap.Int("WTV", len(cfg.WTV.Channels)),
	)

	logger.Info("Initializing state storage...")
	stateStorage, err := storage.NewStateStorage(
		logger,
		filepath.Join(
			exeDir,
			"state.db",
		),
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
		platforms := buildPlatforms(logger, cfg, false)

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
								URL:          streamURL(state.Platform, state.Channel),
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

	if err := notificationService.Send(
		notifier.Notification{
			Title:   "tway",
			Message: "Initializing services and connecting to streaming platforms...",
			Icon:    *iconPath,
		},
	); err != nil {
		logger.Error(
			"Send initialize notify error",
			zap.Error(err),
		)
		return
	}

	logger.Info("Creating notify context...")
	ctx, stop := signal.NotifyContext(
		context.Background(),
		os.Interrupt,
		syscall.SIGTERM,
	)

	defer stop()
	group, ctx := errgroup.WithContext(ctx)

	logger.Info("Notify context created!")

	logger.Info("Starting config reloader...")
	reloader := NewConfigReloader(
		ctx,
		*iconPath,
		logger,
		stateStorage,
		notificationService,
	)

	if err := reloader.Start(cfg); err != nil {
		logger.Error(
			"Start config reloader",
			zap.Error(err),
		)
		return
	}

	logger.Info("Config reloader started!")

	logger.Info("Creating config watcher...")
	configChanges, configErrors := config.Watch(
		ctx,
		*configPath,
		500*time.Millisecond,
	)

	configReloads := make(chan *config.Config, 1)
	group.Go(
		func() error {
			return runConfigWatcher(
				ctx,
				logger,
				*configPath,
				configChanges,
				configErrors,
				configReloads,
			)
		},
	)

	group.Go(
		func() error {
			return runConfigReloads(
				ctx,
				logger,
				configReloads,
				reloader,
			)
		},
	)

	if err := notificationService.Send(
		notifier.Notification{
			Title:   "tway",
			Message: "Initialization completed. All services are connected and stream monitoring is active.",
			Icon:    *iconPath,
		},
	); err != nil {
		logger.Error(
			"Send ready notify error",
			zap.Error(err),
		)
	}

	var refreshRunning atomic.Bool
	var refreshGroup sync.WaitGroup

	logger.Info("Creating tray...")
	trayApp := tray.NewTray(
		logger,
		func() {
			if !refreshRunning.CompareAndSwap(false, true) {
				logger.Info("Manual stream refresh is already running!")

				if err := notificationService.Send(
					notifier.Notification{
						Title:   "tway",
						Message: "Manual stream refresh is already running!",
						Icon:    *iconPath,
					},
				); err != nil {
					logger.Error(
						"Failed to send refresh notification",
						zap.Error(err),
					)
				}
				return
			}

			refreshGroup.Add(1)
			go func() {
				defer refreshGroup.Done()
				defer refreshRunning.Store(false)

				logger.Info("Manual stream refresh requested...")

				if err := notificationService.Send(
					notifier.Notification{
						Title:   "tway",
						Message: "Processing streams status refresh started!",
						Icon:    *iconPath,
					},
				); err != nil {
					logger.Error(
						"Failed to send refresh notification",
						zap.Error(err),
					)
				}

				reloader.WithCurrentPlatforms(
					func(platforms []Platform) {
						initializeStreamStates(
							logger,
							platforms,
							stateStorage,
						)
					},
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
			}()
		},
		func() {
			logger.Info("Manual show streams summary requested!")
			processOverall(
				*iconPath,
				logger,
				stateStorage,
				notificationService,
			)

			logger.Info("Manual show streams summary completed!")
		},
		func() {
			logger.Info("Tray settings requested!")
			if err := tui.OpenTerminal("--setup"); err != nil {
				logger.Error(
					"Open settings terminal",
					zap.Error(err),
				)
			}
		},
		func() {
			logger.Info("Tray exit event has requested!")
			stop()
		},
	)

	logger.Info("Tray created!")
	trayApp.Run()

	stop()
	if err := group.Wait(); err != nil {
		logger.Error(
			"Config worker group stopped with error",
			zap.Error(err),
		)
	}

	reloader.Stop()
	refreshGroup.Wait()
	logger.Info("Tway stopped!")
}
