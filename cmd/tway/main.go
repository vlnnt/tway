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

	"tway/internal/config"
	"tway/internal/logging"
	"tway/internal/notifier"
	"tway/internal/storage"
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

	bootstrapSetupMode := flag.Bool(
		"bootstrap-setup",
		false,
		"Open bootstrap configuration setup",
	)

	afterSetupMode := flag.Bool(
		"after-setup",
		false,
		"Run after setup save",
	)

	logsMode := flag.Bool(
		"logs",
		false,
		"Show logs",
	)

	flag.Parse()
	showStreamersAfterSetup := *afterSetupMode

	var (
		logger  *zap.Logger
		logPath string
	)

	if *logsMode {
		logPath, err := logging.Path()
		if err != nil {
			return
		}

		if err := tui.RunLogs(logPath); err != nil {
			return
		}

		return
	}

	if *tuiMode || *setupMode || *bootstrapSetupMode {
		logger = zap.NewNop()
	} else {
		logger, logPath, err = logging.New()
		if err != nil {
			return
		}

		defer logger.Sync()
	}

	if logPath != "" {
		logger.Info(
			"Logging initialized",
			zap.String("Path", logPath),
		)
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

	if firstRun || *setupMode || *bootstrapSetupMode {
		stop := runSetup(
			logger,
			firstRun,
			setupMode,
			configPath,
			cfg,
			bootstrapSetupMode,
		)

		if stop {
			return
		}

		if firstRun {
			showStreamersAfterSetup = true
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

	if !*tuiMode {
		instanceLock, stop := acquireInstanceLock(
			iconPath,
			logger,
		)

		if stop {
			return
		}

		defer func() {
			if err := instanceLock.Close(); err != nil {
				logger.Error(
					"Release instance lock",
					zap.Error(err),
				)
			}
		}()
	}

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
		runStreamersTUI(
			cfg,
			logger,
			stateStorage,
		)
		return
	}

	logger.Info("Initializing notifier service...")
	notificationService, err := notifier.NewNotifier(logger)
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

	if err := reloader.start(cfg); err != nil {
		logger.Error(
			"Start config reloader",
			zap.Error(err),
		)
		return
	}

	logger.Info("Config reloader started!")

	if showStreamersAfterSetup && cfg.UI.ShowStreamers {
		if err := tui.OpenTerminal("--tui"); err != nil {
			logger.Error(
				"Open streamers after setup",
				zap.Error(err),
			)
		}
	}

	logger.Info("Creating config watcher...")
	configChanges, configErrors := config.Watch(
		ctx,
		*configPath,
		500*time.Millisecond,
	)

	configReloads := make(chan *config.Config, 1)
	group.Go(
		func() error {
			return runConfigWatchLoop(
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
			return runConfigReloadLoop(
				ctx,
				logger,
				configReloads,
				reloader,
			)
		},
	)

	if err := notificationService.Send(
		notifier.Notification{
			Title: "tway",
			Message: "Initialization completed. " +
				"All services are connected and stream monitoring is active.",
			Icon: *iconPath,
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
	trayApp := createTray(
		iconPath,
		logPath,
		logger,
		stop,
		reloader,
		&refreshRunning,
		&refreshGroup,
		stateStorage,
		notificationService,
	)

	logger.Info("Tray created!")

	group.Go(
		func() error {
			return reloader.runFatalHandler(
				ctx,
				logger,
				notificationService,
				*iconPath,
				stop,
				trayApp,
			)
		},
	)

	trayApp.Run()
	stop()

	if err := group.Wait(); err != nil {
		logger.Error(
			"Config worker group stopped with error",
			zap.Error(err),
		)
	}

	reloader.stop()
	refreshGroup.Wait()
	logger.Info("Tway stopped!")
}
