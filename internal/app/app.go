package app

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"golang.org/x/sync/errgroup"

	"tway/internal/config"
	"tway/internal/notifier"
	"tway/internal/twitch"
)

type App struct {
	config   *config.Config
	twitch   *twitch.Client
	notifier notifier.Notifier
	storage  *StateStorage
}

func New(
	cfg *config.Config,
	twitchClient *twitch.Client,
	notificationService notifier.Notifier,
	storage *StateStorage,
) *App {
	return &App{
		config:   cfg,
		twitch:   twitchClient,
		notifier: notificationService,
		storage:  storage,
	}
}

func (a *App) Run(
	ctx context.Context,
) error {
	log.Println("Checking initial stream states...")
	states, err := a.storage.Load()
	if err != nil {
		return fmt.Errorf("load states: %w", err)
	}

	var status strings.Builder
	status.WriteString("Статус стримеров:\n\n")

	for _, streamer := range a.config.Streamers {
		if !streamer.Enabled {
			continue
		}

		channel := streamer.Channel
		stream, err := a.twitch.GetStream(channel)
		if err != nil {
			log.Printf("[%s] Initial check failed: %v", channel, err)
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
		log.Printf("Save initial states failed: %v", err)
	}

	if status.Len() > len("Статус стримеров:\n\n") {
		err := a.notifier.Send(notifier.Notification{
			Title:   "Twitch Watcher",
			Message: status.String(),
			Icon:    "",
		})

		if err != nil {
			log.Printf("Initial notification failed: %v", err)
		}
	}

	log.Println("Starting stream watchers...")
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
						log.Printf("[%s] Watcher stopped", channel)
						return nil

					case <-ticker.C:
						log.Printf("[%s] Checking stream...", channel)
						stream, err := a.twitch.GetStream(channel)
						if err != nil {
							log.Printf("[%s] Twitch check failed: %v", channel, err)
							continue
						}

						log.Printf(
							"[%s] Live=%t Title=%q Game=%q",
							channel,
							stream.IsLive,
							stream.Title,
							stream.Game,
						)

						if !wasLive && stream.IsLive {
							log.Printf("[%s] Stream started", channel)
							err := a.notifier.Send(notifier.Notification{
								Title: channel + " начал трансляцию",
								Message: fmt.Sprintf(
									"%s\nКатегория: %s",
									stream.Title,
									stream.Game,
								),
								Icon: "",
								URL:  "https://twitch.tv/" + channel,
							})
							if err != nil {
								log.Printf("[%s] Notify failed: %v", channel, err)
							}

							wasLive = true
						}

						if wasLive && !stream.IsLive {
							log.Printf("[%s] Stream ended", channel)
							err := a.notifier.Send(notifier.Notification{
								Title:   channel + " закончил трансляцию",
								Message: "Стример вышел из эфира",
								Icon:    "",
								URL:     "https://twitch.tv/" + channel,
							})
							if err != nil {
								log.Printf("[%s] Notify failed: %v", channel, err)
							}

							wasLive = false
						}

						states[channel] = StreamState{
							IsLive:    stream.IsLive,
							StreamID:  stream.ID,
							UpdatedAt: time.Now(),
						}

						if err := a.storage.Save(states); err != nil {
							log.Printf("[%s] Save state failed: %v", channel, err)
						}
					}
				}
			}
		}(channel))
	}

	log.Println("All watchers started!")
	return group.Wait()
}
