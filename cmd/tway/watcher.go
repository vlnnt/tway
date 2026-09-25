package main

import (
	"context"
	"tway/internal/config"
	"tway/internal/tui"

	"go.uber.org/zap"
)

func runConfigWatchLoop(
	ctx context.Context,
	logger *zap.Logger,
	configPath string,
	configChanges <-chan struct{},
	configErrors <-chan error,
	configReloads chan<- *config.Config,
) error {
	for {
		select {
		case <-ctx.Done():
			return nil

		case _, ok := <-configChanges:
			if !ok {
				configChanges = nil
				if configErrors == nil {
					return nil
				}

				continue
			}

			newConfig, err := config.LoadConfig(configPath)
			if err != nil {
				logger.Error(
					"Reload config",
					zap.Error(err),
				)
				continue
			}

			select {
			case configReloads <- newConfig:
			case <-ctx.Done():
				return nil
			}

		case err, ok := <-configErrors:
			if !ok {
				configErrors = nil
				if configChanges == nil {
					return nil
				}

				continue
			}

			logger.Error(
				"Config watcher",
				zap.Error(err),
			)
		}
	}
}

func runConfigReloadLoop(
	ctx context.Context,
	logger *zap.Logger,
	configReloads <-chan *config.Config,
	reloader *ConfigReloader,
) error {
	for {
		select {
		case <-ctx.Done():
			return nil

		case newConfig, ok := <-configReloads:
			if !ok {
				return nil
			}

			logger.Info("Config hot reload started...")
			if err := reloader.reload(newConfig); err != nil {
				logger.Error(
					"Config hot reload failed",
					zap.Error(err),
				)
				continue
			}

			logger.Info("Config hot reload complete!")
			if newConfig.UI.ShowStreamers {
				if err := tui.OpenTerminal("--tui"); err != nil {
					logger.Error(
						"Open streamers after config save",
						zap.Error(err),
					)
				}
			}
		}
	}
}
