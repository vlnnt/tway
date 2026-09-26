package main

import (
	"context"
	"fmt"
	"sync"
	"time"
	"tway/internal/app"
	"tway/internal/config"
	"tway/internal/notifier"
	"tway/internal/storage"
	"tway/internal/tray"

	"go.uber.org/zap"
)

type ConfigReloader struct {
	ctx                 context.Context
	icon                string
	logger              *zap.Logger
	stateStorage        *storage.StateStorage
	notificationService notifier.Notifier
	mu                  sync.RWMutex
	cfg                 *config.Config
	summaryInterval     time.Duration
	platforms           []Platform
	applications        []*app.App
	workers             *workerSet
	fatalErrors         chan error
}

func NewConfigReloader(
	ctx context.Context,
	icon string,
	logger *zap.Logger,
	stateStorage *storage.StateStorage,
	notificationService notifier.Notifier,
) *ConfigReloader {
	return &ConfigReloader{
		ctx:                 ctx,
		icon:                icon,
		logger:              logger,
		stateStorage:        stateStorage,
		notificationService: notificationService,
		fatalErrors:         make(chan error, 1),
	}
}

func (cr *ConfigReloader) start(
	cfg *config.Config,
) error {
	cr.mu.RLock()
	started := cr.workers != nil
	cr.mu.RUnlock()

	if started {
		return fmt.Errorf("config reloader already started")
	}

	checkInterval, summaryInterval, err := parseIntervals(cfg)
	if err != nil {
		return err
	}

	platforms := platformsFromConfig(
		cr.logger,
		cfg,
		true,
	)

	if err := syncTrackedStreams(
		cr.logger,
		platforms,
		cr.stateStorage,
	); err != nil {
		return err
	}

	refreshStreamStates(
		cr.logger,
		platforms,
		cr.stateStorage,
	)

	applications := createApplications(
		cr.icon,
		cr.logger,
		platforms,
		checkInterval,
		cr.notificationService,
		cr.stateStorage,
	)

	cr.mu.Lock()
	defer cr.mu.Unlock()

	if cr.workers != nil {
		return fmt.Errorf("config reloader already started")
	}

	workers := startWorkers(
		cr.ctx,
		cr.icon,
		cr.logger,
		cfg,
		platforms,
		applications,
		summaryInterval,
		cr.stateStorage,
		cr.notificationService,
		cr.reportFatal,
	)

	cr.cfg = cfg
	cr.summaryInterval = summaryInterval
	cr.platforms = platforms
	cr.applications = applications
	cr.workers = workers

	cr.logger.Info(
		"Config reloader started",
		zap.Duration(
			"Check interval",
			checkInterval,
		),
		zap.Duration(
			"Summary interval",
			summaryInterval,
		),
		zap.Bool(
			"Summary enabled",
			cfg.Summary.Enable,
		),
		zap.Int(
			"Platforms",
			len(platforms),
		),
	)

	return nil
}

func (cr *ConfigReloader) reload(
	cfg *config.Config,
) error {
	checkInterval, summaryInterval, err := parseIntervals(cfg)
	if err != nil {
		return err
	}

	platforms := platformsFromConfig(
		cr.logger,
		cfg,
		true,
	)

	applications := createApplications(
		cr.icon,
		cr.logger,
		platforms,
		checkInterval,
		cr.notificationService,
		cr.stateStorage,
	)

	cr.mu.Lock()
	defer cr.mu.Unlock()

	if cr.workers == nil {
		return fmt.Errorf("config reloader isn't started")
	}

	oldConfig := cr.cfg
	oldPlatforms := cr.platforms
	oldApplications := cr.applications
	oldSummaryInterval := cr.summaryInterval

	cr.logger.Info("Stopping config workers...")
	cr.workers.stopAndWait()

	cr.logger.Info("Config workers stopped!")

	if err := syncTrackedStreams(
		cr.logger,
		platforms,
		cr.stateStorage,
	); err != nil {
		cr.logger.Error(
			"Failed to synchronize reloaded config",
			zap.Error(err),
		)

		cr.workers = startWorkers(
			cr.ctx,
			cr.icon,
			cr.logger,
			oldConfig,
			oldPlatforms,
			oldApplications,
			oldSummaryInterval,
			cr.stateStorage,
			cr.notificationService,
			cr.reportFatal,
		)

		return err
	}

	refreshStreamStates(
		cr.logger,
		platforms,
		cr.stateStorage,
	)

	workers := startWorkers(
		cr.ctx,
		cr.icon,
		cr.logger,
		cfg,
		platforms,
		applications,
		summaryInterval,
		cr.stateStorage,
		cr.notificationService,
		cr.reportFatal,
	)

	cr.cfg = cfg
	cr.summaryInterval = summaryInterval
	cr.platforms = platforms
	cr.applications = applications
	cr.workers = workers

	cr.logger.Info(
		"Config reloaded",
		zap.Duration(
			"Check interval",
			checkInterval,
		),
		zap.Duration(
			"Summary interval",
			summaryInterval,
		),
		zap.Bool(
			"Summary enable",
			cfg.Summary.Enable,
		),
		zap.Int(
			"Platforms",
			len(platforms),
		),
	)

	return nil
}

func (cr *ConfigReloader) withPlatforms(
	fn func([]Platform),
) {
	cr.mu.RLock()
	defer cr.mu.RUnlock()

	fn(cr.platforms)
}

func (cr *ConfigReloader) stop() {
	cr.mu.Lock()
	defer cr.mu.Unlock()

	if cr.workers == nil {
		return
	}

	cr.logger.Info("Stopping config reloader...")

	cr.workers.stopAndWait()
	cr.workers = nil

	cr.logger.Info("Config reloader stopped!")
}

func (cr *ConfigReloader) runFatalHandler(
	ctx context.Context,
	logger *zap.Logger,
	notificationService notifier.Notifier,
	iconPath string,
	stop context.CancelFunc,
	trayApp *tray.Tray,
) error {
	select {
	case <-ctx.Done():
		return nil

	case err := <-cr.fatal():
		logger.Error(
			"Monitoring stopped with fatal error",
			zap.Error(err),
		)

		if notifyErr := notificationService.Send(
			notifier.Notification{
				Title: "tway",
				Message: "Stream monitoring stopped due to a fatal error. " +
					"Tway will be closed.",
				Icon: iconPath,
			},
		); notifyErr != nil {
			logger.Error(
				"Send fatal monitoring notification",
				zap.Error(notifyErr),
			)
		}
	}

	stop()
	trayApp.Quit()

	return nil
}

func (cr *ConfigReloader) reportFatal(
	err error,
) {
	select {
	case cr.fatalErrors <- err:
	default:
	}
}

func (cr *ConfigReloader) fatal() <-chan error {
	return cr.fatalErrors
}

func parseIntervals(
	cfg *config.Config,
) (time.Duration, time.Duration, error) {
	checkInterval, err := time.ParseDuration(cfg.Check)
	if err != nil {
		return 0, 0, fmt.Errorf("parse check interval: %w", err)
	}

	if checkInterval <= 0 {
		return 0, 0, fmt.Errorf(
			"check interval must be greater zero: %s", checkInterval)
	}

	var summaryInterval time.Duration
	if cfg.Summary.Enable {
		summaryInterval, err = time.ParseDuration(cfg.Summary.Interval)
		if err != nil {
			return 0, 0, fmt.Errorf("parse summary interval: %w", err)
		}

		if summaryInterval <= 0 {
			return 0, 0, fmt.Errorf(
				"summary interval must be greater than zero: %s",
				summaryInterval,
			)
		}
	}

	return checkInterval, summaryInterval, nil
}
