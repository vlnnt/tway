package main

import (
	"errors"
	"tway/internal/i18n"
	"tway/internal/instance"
	"tway/internal/notifier"

	"go.uber.org/zap"
)

func acquireInstanceLock(
	iconPath *string,
	logger *zap.Logger,
	texts i18n.Texts,
) (*instance.Lock, bool) {
	instanceLock, err := instance.Acquire()
	if err != nil {
		if errors.Is(err, instance.ErrAlreadyRunning) {
			logger.Info("Tway is already running!")
			notificationService, notifyErr := notifier.NewNotifier(logger)
			if notifyErr != nil {
				logger.Error(
					"Create notifier",
					zap.Error(notifyErr),
				)
				return nil, true
			}

			if notifyErr := notificationService.Send(
				notifier.Notification{
					Title:   "tway",
					Message: texts.StreamSingleInstanceMessage,
					Icon:    *iconPath,
				},
			); notifyErr != nil {
				logger.Error(
					"Send already running notification",
					zap.Error(notifyErr),
				)
			}

			notificationService.Close()
			return nil, true
		}

		logger.Error(
			"Acquire instance lock",
			zap.Error(err),
		)
		return nil, true
	}

	return instanceLock, false
}
