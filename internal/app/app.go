package app

import (
	"context"
	"fmt"
	"time"

	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"

	"tway/internal/config"
	"tway/internal/notifier"
	"tway/internal/twitch"
)

type App struct {
	icon     string
	log      *zap.Logger
	config   *config.Config
	twitch   *twitch.Client
	notifier notifier.Notifier
	storage  *StateStorage
}

func NewApp(
	icon string,
	log *zap.Logger,
	cfg *config.Config,
	twitchClient *twitch.Client,
	notificationService notifier.Notifier,
	storage *StateStorage,
) *App {
	return &App{
		icon:     icon,
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
	a.log.Info("App.Run",
		zap.String("Run", "Starting the stream workers ..."))
	a.sendOverallStatus()

	group, ctx := errgroup.WithContext(ctx)
	group.Go(func() error {
		ticker := time.NewTicker(5 * time.Minute)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				a.log.Info(
					"App.Run.ctx.Done",
					zap.String("Worker", "Status summary"),
				)
				return nil

			case <-ticker.C:
				a.log.Info(
					"App.Run.statusTicker",
					zap.String("Run", "Updating overall stream status ..."),
				)

				a.sendOverallStatus()
			}
		}
	})

	for _, channel := range a.config.Streamers {
		group.Go(func(channel string) func() error {
			return func() error {
				ticker := time.NewTicker(
					a.config.CheckInterval.Duration,
				)

				defer ticker.Stop()
				state, err := a.storage.Get(channel)
				if err != nil {
					a.log.Error("App.Run.GetState", zap.Error(err))
					return err
				}

				wasLive := false
				if state != nil {
					wasLive = state.IsLive
				}

				for {
					select {
					case <-ctx.Done():
						a.log.Info("App.Run.ctx.Done",
							zap.String("Worker stopped:", channel))
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
								Icon: a.icon,
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
								Icon:    a.icon,
								URL:     "https://twitch.tv/" + channel,
							})
							if err != nil {
								a.log.Error("Notify failed",
									zap.String("Channel", channel), zap.Error(err))
							}

							wasLive = false
						}

						err = a.storage.Save(StreamState{
							Channel:   channel,
							IsLive:    stream.IsLive,
							StreamID:  stream.ID,
							UpdatedAt: time.Now(),
						})
						if err != nil {
							a.log.Error(
								"Save state failed",
								zap.String("Channel", channel),
								zap.Error(err),
							)
						}
					}
				}
			}
		}(channel))
	}

	a.log.Info("App.Run",
		zap.String("Run", "All stream workers are running!"))
	return group.Wait()
}

func (a *App) sendOverallStatus() {
	a.log.Info("App.sendOverallStatus",
		zap.String("sendOverallStatus",
			"Processing summary stream status ..."))
	online, offline := 0, 0
	for _, channel := range a.config.Streamers {
		stream, err := a.twitch.GetStream(channel)
		if err != nil {
			a.log.Error(
				"App.sendOverallStatus.GetStream",
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

	if online+offline == 0 {
		a.log.Warn(
			"App.sendOverallStatus",
			zap.String("Run", "No stream statuses received"),
		)
		return
	}

	status := fmt.Sprintf(
		"🟢 Online: %d\n🔴 Offline: %d",
		online,
		offline,
	)

	if err := a.notifier.Send(
		notifier.Notification{
			Title:   "tway",
			Message: status,
			Icon:    a.icon,
		},
	); err != nil {
		a.log.Error(
			"App.sendOverallStatus.Send",
			zap.Error(err),
		)
	}
	a.log.Info("App.sendOverallStatus",
		zap.String("sendOverallStatus",
			"Summary stream status processed!"))
}
