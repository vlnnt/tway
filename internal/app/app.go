package app

import (
	"context"
	"fmt"
	"log"
	"time"
	"tway/internal/config"
	"tway/internal/notifier"
	"tway/internal/twitch"

	"golang.org/x/sync/errgroup"
)

type App struct {
	config   *config.Config
	twitch   *twitch.Client
	notifier notifier.Notifier
}

func NewApp(
	config *config.Config,
	twitchClient *twitch.Client,
	notificationService notifier.Notifier,
) *App {
	return &App{
		config:   config,
		twitch:   twitchClient,
		notifier: notificationService,
	}
}

func (a *App) Run(
	ctx context.Context,
) error {
	group, ctx := errgroup.WithContext(ctx)
	for _, streamer := range a.config.Streamers {
		if !streamer.Enabled {
			continue
		}

		channel := streamer.Channel
		group.Go(func() error {
			ticker := time.NewTicker(a.config.CheckInterval.Duration)
			defer ticker.Stop()

			var wasLive bool
			var initialized bool

			for {
				stream, err := a.twitch.GetStream(channel)
				if err != nil {
					log.Printf("check channel %q: %v", channel, err)
				} else {
					if initialized && !wasLive && stream.IsLive {
						err = a.notifier.Send(notifier.Notification{
							Title: channel + " начал трансляцию",
							Message: fmt.Sprintf(
								"%s\nКатегория: %s",
								stream.Title,
								stream.Game,
							),
							Icon: "../icon/logo.png",
							URL:  "https://twitch.tv/" + channel,
						})
						if err != nil {
							log.Printf("send notification for %q: %v", channel, err)
						}
					}

					wasLive = stream.IsLive
					initialized = true
				}

				select {
				case <-ctx.Done():
					return nil

				case <-ticker.C:
				}
			}
		})
	}

	return group.Wait()
}
