package main

import (
	"context"
	"fmt"
	"time"

	"tway/internal/notifier"
	"tway/internal/storage"

	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
)

const streamInitConcurrency = 4

func ensureStreamStates(
	logger *zap.Logger,
	platforms []Platform,
	stateStorage *storage.StateStorage,
) error {
	logger.Info("Synchronizing tracked stream states...")
	active := make([]storage.StreamKey, 0)

	for _, platform := range platforms {
		for _, channel := range platform.Channels {
			active = append(
				active,
				storage.StreamKey{
					Platform: platform.Name,
					Channel:  channel,
				},
			)
		}
	}

	if err := stateStorage.SyncTracked(active); err != nil {
		logger.Error(
			"Failed to synchronize tracked stream states",
			zap.Error(err),
		)

		return err
	}

	logger.Info(
		"Tracked stream states synchronized!",
		zap.Int(
			"Tracked",
			len(active),
		),
	)

	return nil
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

	var group errgroup.Group
	group.SetLimit(streamInitConcurrency)

	for _, platform := range platforms {
		for _, channel := range platform.Channels {
			platform := platform
			channel := channel
			group.Go(func() error {
				stream, err := platform.Client.GetStream(channel)
				if err != nil {
					logger.Error(
						"Failed to get initial stream state",
						zap.String("Platform", platform.Name),
						zap.String("Channel", channel),
						zap.Error(err),
					)

					return nil
				}

				currentState, err := stateStorage.Get(platform.Name, channel)
				if err != nil {
					logger.Error(
						"Failed to get stored stream state",
						zap.String("Platform", platform.Name),
						zap.String("Channel", channel),
						zap.Error(err),
					)

					return nil
				}

				lastStreamAt := time.Time{}
				startedAt := time.Time{}

				if currentState != nil {
					lastStreamAt = currentState.LastStreamAt
				}

				if !stream.LastStreamAt.IsZero() {
					lastStreamAt = stream.LastStreamAt
				}

				if stream.IsLive {
					if !stream.StartedAt.IsZero() {
						startedAt = stream.StartedAt
					} else if currentState != nil &&
						currentState.IsLive {
						startedAt = currentState.StartedAt
					}
				}

				err = stateStorage.Update(
					storage.StreamState{
						Platform:     platform.Name,
						Channel:      channel,
						IsTracked:    true,
						IsLive:       stream.IsLive,
						LastStreamAt: lastStreamAt,
						StartedAt:    startedAt,
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

				return nil
			})
		}
	}

	_ = group.Wait()
	logger.Info("Stream states initialized!")
}

func processOverall(
	icon string,
	logger *zap.Logger,
	state *storage.StateStorage,
	notificationService notifier.Notifier,
) {
	logger.Info("Processing overall streams...")
	states, err := state.GetTracked()
	if err != nil {
		logger.Error(
			"Failed to get tracked stream states",
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
		logger.Warn("No tracked stream statuses received!")
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
