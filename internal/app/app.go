package app

import (
	"context"
	"fmt"
	"strings"
	"time"

	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"

	"tway/internal/config"
	"tway/internal/notifier"
	"tway/internal/twitch"
)

type App struct {
	log      *zap.Logger
	config   *config.Config
	twitch   *twitch.Client
	notifier notifier.Notifier
	storage  *StateStorage
}

func New(
	log *zap.Logger,
	cfg *config.Config,
	twitchClient *twitch.Client,
	notificationService notifier.Notifier,
	storage *StateStorage,
) *App {
	return &App{
		log:      log,
		config:   cfg,
		twitch:   twitchClient,
		notifier: notificationService,
		storage:  storage,
	}
}

func (a *App) Run(
	ctx context.Context,
) error {
	a.log.Info("Checking initial stream states ...")
	states, err := a.storage.Load()
	if err != nil {
		a.log.Error("App.Run.Load", zap.Error(err))
		return err
	}

	var status strings.Builder
	status.WriteString("Streamers's statuses:\n\n")

	for _, streamer := range a.config.Streamers {
		if !streamer.Enabled {
			continue
		}

		channel := streamer.Channel
		stream, err := a.twitch.GetStream(channel)
		if err != nil {
			a.log.Error("App.Run.GetStream", zap.Error(err))
			continue
		}

		states[channel] = StreamState{
			IsLive:    stream.IsLive,
			StreamID:  stream.ID,
			UpdatedAt: time.Now(),
		}

		if stream.IsLive {
			status.WriteString(fmt.Sprintf("🟢 %s — LIVE\n", channel))
		} else {
			status.WriteString(fmt.Sprintf("🔴 %s — OFFLINE\n", channel))
		}
	}

	if err := a.storage.Save(states); err != nil {
		a.log.Error("App.Run.Save", zap.Error(err))
	}

	if status.Len() > len("Streamers's statuses:\n\n") {
		err := a.notifier.Send(notifier.Notification{
			Title:   "tway",
			Message: status.String(),
			Icon:    "",
		})

		if err != nil {
			a.log.Error("App.Run.Send", zap.Error(err))
		}
	}

	a.log.Info("App.Run",
		zap.String("Run", "Starting the stream watchers ..."))
	group, ctx := errgroup.WithContext(ctx)
	for _, streamer := range a.config.Streamers {
		if !streamer.Enabled {
			continue
		}

		channel := streamer.Channel
		group.Go(func(channel string) func() error {
			return func() error {
				ticker := time.NewTicker(
					a.config.CheckInterval.Duration,
				)

				defer ticker.Stop()
				wasLive := states[channel].IsLive

				for {
					select {
					case <-ctx.Done():
						a.log.Info("App.Run.ctx.Done",
							zap.String("Watcher stopped:", channel))
						return nil

					case <-ticker.C:
						a.log.Info("App.Run.ticket.C",
							zap.String("Checking stream ...", channel))
						stream, err := a.twitch.GetStream(channel)
						if err != nil {
							a.log.Error("App.Run.GetStream", zap.Error(err))
							continue
						}

						a.log.Info("State",
							zap.String("Channel", channel),
							zap.Bool("Live", stream.IsLive),
							zap.String("Title", stream.Title),
							zap.String("Game", stream.Game),
						)

						if !wasLive && stream.IsLive {
							a.log.Info("Stream started",
								zap.String("Channel", channel))
							err := a.notifier.Send(notifier.Notification{
								Title: channel + " has started the broadcast",
								Message: fmt.Sprintf(
									"%s\nCategory: %s",
									stream.Title,
									stream.Game,
								),
								Icon: "",
								URL:  "https://twitch.tv/" + channel,
							})
							if err != nil {
								a.log.Error("Notify failed",
									zap.String("Channel", channel), zap.Error(err))
							}

							wasLive = true
						}

						if wasLive && !stream.IsLive {
							a.log.Info("The stream has ended",
								zap.String("Channel", channel))
							err := a.notifier.Send(notifier.Notification{
								Title:   channel + " has ended the broadcast",
								Message: "The streamer has left the broadcast",
								Icon:    "",
								URL:     "https://twitch.tv/" + channel,
							})
							if err != nil {
								a.log.Error("Notify failed",
									zap.String("Channel", channel), zap.Error(err))
							}

							wasLive = false
						}

						states[channel] = StreamState{
							IsLive:    stream.IsLive,
							StreamID:  stream.ID,
							UpdatedAt: time.Now(),
						}

						if err := a.storage.Save(states); err != nil {
							a.log.Error("Save state failed",
								zap.String("Channel", channel), zap.Error(err))
						}
					}
				}
			}
		}(channel))
	}

	a.log.Info("App.Run",
		zap.String("Run", "All the watchers have started!"))
	return group.Wait()
}
