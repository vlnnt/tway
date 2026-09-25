package main

import (
	"context"
	"tway/internal/config"

	"go.uber.org/zap"
)

func runConfigWatcher(
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

func runConfigReloads(
	ctx context.Context,
	logger *zap.Logger,
	configReloads <-chan *config.Config,
	reloader *ConfigReloader,
) error {
	for {
		select {
		case <-ctx.Done():
			return nil

		case newConfig := <-configReloads:
			logger.Info("Config hot reload started...")
			if err := reloader.Reload(newConfig); err != nil {
				logger.Error(
					"Config hot reload failed",
					zap.Error(err),
				)

				continue
			}

			logger.Info("Config hot reload complete!")
		}
	}
}
