package app

import (
	"context"
	"fmt"
	"time"

	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"

	"tway/internal/client"
	"tway/internal/notifier"
	"tway/internal/storage"
)

type App struct {
	icon          string
	log           *zap.Logger
	platform      string
	channels      []string
	checkInterval time.Duration
	client        client.Client
	notifier      notifier.Notifier
	storage       *storage.StateStorage
}

func NewApp(
	icon string,
	log *zap.Logger,
	platform string,
	channels []string,
	checkInterval time.Duration,
	client client.Client,
	notificationService notifier.Notifier,
	storage *storage.StateStorage,
) *App {
	return &App{
		icon:          icon,
		log:           log,
		platform:      platform,
		channels:      channels,
		checkInterval: checkInterval,
		client:        client,
		notifier:      notificationService,
		storage:       storage,
	}
}

func (a *App) Run(
	ctx context.Context,
) error {
	a.log.Info(
		"Starting platform worker",
		zap.String("Platform", a.platform),
		zap.Int("Channels", len(a.channels)),
		zap.Duration("Check interval", a.checkInterval),
	)

	group, ctx := errgroup.WithContext(ctx)

	for _, channel := range a.channels {
		group.Go(func(channel string) func() error {
			return func() error {
				ticker := time.NewTicker(a.checkInterval)
				defer ticker.Stop()

				state, err := a.storage.Get(a.platform, channel)
				if err != nil {
					a.log.Error(
						"Failed to load stream state",
						zap.String("Platform", a.platform),
						zap.String("Channel", channel),
						zap.Error(err),
					)
					return err
				}

				wasLive := false
				if state != nil {
					wasLive = state.IsLive
				}

				a.log.Info(
					"Channel worker started",
					zap.String("Platform", a.platform),
					zap.String("Channel", channel),
					zap.Bool("Was live status", wasLive),
				)

				for {
					select {
					case <-ctx.Done():
						a.log.Info(
							"Channel worker stopped",
							zap.String("Platform", a.platform),
							zap.String("Channel", channel),
						)

						return nil

					case <-ticker.C:
						a.log.Info(
							"Checking stream",
							zap.String("Platform", a.platform),
							zap.String("Channel", channel),
						)

						stream, err := a.client.GetStream(channel)
						if err != nil {
							a.log.Error(
								"Failed to get stream",
								zap.String("Platform", a.platform),
								zap.String("Channel", channel),
								zap.Error(err),
							)

							continue
						}

						a.log.Info(
							"Stream status updated",
							zap.String("Platform", a.platform),
							zap.String("Channel", channel),
							zap.Bool("Live", stream.IsLive),
							zap.String("Title", stream.Title),
							zap.String("Subcategory", stream.Subcategory),
						)

						if !wasLive && stream.IsLive {
							a.log.Info(
								"Stream started",
								zap.String("Platform", a.platform),
								zap.String("Channel", channel),
							)

							err := a.notifier.Send(
								notifier.Notification{
									Title: channel + " is now live!",
									Message: fmt.Sprintf(
										"%s\nCategory: %s",
										stream.Title,
										stream.Subcategory,
									),
									Icon: a.icon,
									URL:  stream.URL,
								},
							)

							if err != nil {
								a.log.Error(
									"Failed to send stream start notification",
									zap.String("Platform", a.platform),
									zap.String("Channel", channel),
									zap.Error(err),
								)
							}

							wasLive = true
						}

						if wasLive && !stream.IsLive {
							a.log.Info(
								"Stream ended",
								zap.String("Platform", a.platform),
								zap.String("Channel", channel),
							)

							err := a.notifier.Send(
								notifier.Notification{
									Title:   channel + " is no longer live!",
									Message: "The streamer has left the broadcast!",
									Icon:    a.icon,
									URL:     stream.URL,
								},
							)

							if err != nil {
								a.log.Error(
									"Failed to send stream end notification",
									zap.String("Platform", a.platform),
									zap.String("Channel", channel),
									zap.Error(err),
								)
							}

							wasLive = false
						}

						err = a.storage.Update(
							storage.StreamState{
								Platform:  a.platform,
								Channel:   channel,
								IsLive:    stream.IsLive,
								StreamID:  stream.ID,
								UpdatedAt: time.Now(),
							},
						)

						if err != nil {
							a.log.Error(
								"Failed to update stream state",
								zap.String("Platform", a.platform),
								zap.String("Channel", channel),
								zap.Error(err),
							)
						}
					}
				}
			}
		}(channel))
	}

	a.log.Info(
		"Platform worker started",
		zap.String("Platform", a.platform),
		zap.Int("Workers", len(a.channels)),
	)

	return group.Wait()
}
