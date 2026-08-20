package main

import (
	"context"
	"fmt"
	"time"

	"tway/internal/client"
	"tway/internal/notifier"
	"tway/internal/storage"

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
	interval time.Duration,
	state *storage.StateStorage,
	notificationService notifier.Notifier,
	icon string,
) {
	processOverall(
		icon,
		logger,
		state,
		notificationService,
	)

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			logger.Info("Summary worker stopped")
			return

		case <-ticker.C:
			processOverall(
				icon,
				logger,
				state,
				notificationService,
			)
		}
	}
}

func initializeStreamStates(
	logger *zap.Logger,
	platforms []Platform,
	stateStorage *storage.StateStorage,
) {
	logger.Info("Initializing stream states...")
	for _, platform := range platforms {
		for _, channel := range platform.Channels {
			stream, err := platform.Client.GetStream(channel)
			if err != nil {
				logger.Error(
					"Failed to get initial stream state",
					zap.String("Platform", platform.Name),
					zap.String("Channel", channel),
					zap.Error(err),
				)
				continue
			}

			err = stateStorage.Update(
				storage.StreamState{
					Platform:  platform.Name,
					Channel:   channel,
					IsLive:    stream.IsLive,
					StreamID:  stream.ID,
					UpdatedAt: time.Now(),
				},
			)
			if err != nil {
				logger.Error(
					"Failed to update initial stream state",
					zap.String("Platform", platform.Name),
					zap.String("Channel", channel),
					zap.Error(err),
				)
			}
		}
	}

	logger.Info("Stream states initialized!")
}

func processOverall(
	icon string,
	logger *zap.Logger,
	state *storage.StateStorage,
	notificationService notifier.Notifier,
) {
	logger.Info("Processing overall streams...")
	states, err := state.GetAll()
	if err != nil {
		logger.Error(
			"Failed to get stream states",
			zap.Error(err),
		)
		return
	}

	online, offline := 0, 0
	for _, stream := range states {
		if stream.IsLive {
			online++
		} else {
			offline++
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
		"Summary notification sent",
		zap.Int("Online", online),
		zap.Int("Offline", offline),
	)
}

func streamURL(
	platform, channel string,
) string {
	switch platform {
	case "twitch":
		return "https://www.twitch.tv/" + channel

	case "kick":
		return "https://kick.com/" + channel

	case "youtube":
		return "https://www.youtube.com/@" + channel

	case "wtv":
		return "https://w.tv/" + channel

	default:
		return ""
	}
}
