package main

import (
	"context"
	"sync"
	"sync/atomic"
	"tway/internal/notifier"
	"tway/internal/storage"
	"tway/internal/tray"
	"tway/internal/tui"

	"go.uber.org/zap"
)

func createTray(
	iconPath *string,
	logPath string,
	logger *zap.Logger,
	stop context.CancelFunc,
	reloader *ConfigReloader,
	refreshRunning *atomic.Bool,
	refreshGroup *sync.WaitGroup,
	stateStorage *storage.StateStorage,
	notificationService notifier.Notifier,
) *tray.Tray {
	trayApp := tray.NewTray(
		logger,
		reloader.currentTexts(),
		func() {
			if !refreshRunning.CompareAndSwap(false, true) {
				logger.Info("Manual stream refresh is already running!")

				texts := reloader.currentTexts()
				if err := notificationService.Send(
					notifier.Notification{
						Title:   "tway",
						Message: texts.StreamManualRefreshAlreadyRunningMessage,
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

				texts := reloader.currentTexts()
				logger.Info("Manual stream refresh requested...")

				if err := notificationService.Send(
					notifier.Notification{
						Title:   "tway",
						Message: texts.StreamManualRefreshRequestMessage,
						Icon:    *iconPath,
					},
				); err != nil {
					logger.Error(
						"Failed to send refresh notification",
						zap.Error(err),
					)
				}

				platforms := reloader.platformsSnapshot()
				refreshStreamStates(
					logger,
					platforms,
					stateStorage,
				)

				if err := notificationService.Send(
					notifier.Notification{
						Title:   "tway",
						Message: texts.StreamManualRefreshStatusReadyMessage,
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
			sendStreamSummary(
				*iconPath,
				logger,
				stateStorage,
				notificationService,
				reloader.currentTexts(),
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
			logger.Info(
				"Open logs requested",
				zap.String("Path", logPath),
			)

			if err := tui.OpenTerminal("--logs"); err != nil {
				logger.Error(
					"Open logs terminal",
					zap.Error(err),
				)
			}
		},
		func() {
			logger.Info("Tray exit event has requested!")
			stop()
		},
	)

	return trayApp
}
