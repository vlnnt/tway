package main

import (
	"context"
	"fmt"
	"time"

	"tway/internal/client"
	"tway/internal/notifier"

	"go.uber.org/zap"
)

type Platform struct {
	Name     string
	Channels []string
	Client   client.Client
}

func runSummaryWorker(
	ctx context.Context,
	logger *zap.Logger,
	platforms []Platform,
	interval time.Duration,
	notificationService notifier.Notifier,
	icon string,
) {
	sendOverallStatus(
		logger,
		platforms,
		notificationService,
		icon,
	)

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			logger.Info("Summary worker stopped")
			return

		case <-ticker.C:
			sendOverallStatus(
				logger,
				platforms,
				notificationService,
				icon,
			)
		}
	}
}

func sendOverallStatus(
	logger *zap.Logger,
	platforms []Platform,
	notificationService notifier.Notifier,
	icon string,
) {
	logger.Info("Processing overall stream status ...")
	online, offline := 0, 0
	for _, platform := range platforms {
		for _, channel := range platform.Channels {
			stream, err := platform.Client.GetStream(channel)
			if err != nil {
				logger.Error(
					"Failed to get stream for summary",
					zap.String("Platform", platform.Name),
					zap.String("Channel", channel),
					zap.Error(err),
				)
				continue
			}

			if stream.IsLive {
				online++
			} else {
				offline++
			}
		}
	}

	if online+offline == 0 {
		logger.Warn("No stream statuses received!")
		return
	}

	status := fmt.Sprintf(
		"🟢 Online: %d\n🔴 Offline: %d",
		online,
		offline,
	)

	if err := notificationService.Send(
		notifier.Notification{
			Title:   "tway",
			Message: status,
			Icon:    icon,
		},
	); err != nil {
		logger.Error(
			"Failed to send summary notification",
			zap.Error(err),
		)
		return
	}

	logger.Info(
		"Overall stream status processed",
		zap.Int("Online", online),
		zap.Int("Offline", offline),
	)
}
