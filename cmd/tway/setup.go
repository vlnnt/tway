package main

import (
	"errors"
	"tway/internal/config"
	"tway/internal/tui"

	"go.uber.org/zap"
)

func runSetup(
	logger *zap.Logger,
	firstRun bool,
	setupMode *bool,
	configPath *string,
	cfg *config.Config,
	bootstrapSetupMode *bool,
) bool {
	if err := tui.AttachConsole(); err != nil {
		if errors.Is(err, tui.ErrNoConsole) {
			if *bootstrapSetupMode {
				return true
			}

			mode := "--setup"
			if firstRun && !*setupMode {
				mode = "--bootstrap-setup"
			}

			if err := tui.OpenTerminal(mode); err != nil {
				logger.Error(
					"Open setup terminal",
					zap.Error(err),
				)
			}
			return true
		}

		logger.Error(
			"tui.AttachConsole",
			zap.Error(err),
		)
		return true
	}

	ui := tui.NewTUI()
	saved, err := ui.ShowSetup(cfg)
	if err != nil {
		logger.Error(
			"Show setup",
			zap.Error(err),
		)
		return true
	}

	if !saved {
		return true
	}

	if err := config.SaveConfig(*configPath, cfg); err != nil {
		logger.Error(
			"config.SaveConfig",
			zap.Error(err),
		)
		return true
	}

	if *bootstrapSetupMode {
		if err := tui.StartDetached("--after-setup"); err != nil {
			logger.Error(
				"Start tway after bootstrap",
				zap.Error(err),
			)
		}
		return true
	}

	if *setupMode {
		return true
	}

	return false
}
