package main

import (
	"context"
	"fmt"
	"sync"
	"time"

	"tway/internal/app"
	"tway/internal/client"
	"tway/internal/client/kick"
	"tway/internal/client/twitch"
	"tway/internal/client/wtv"
	"tway/internal/client/youtube"
	"tway/internal/config"
	"tway/internal/notifier"
	"tway/internal/storage"

	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
)

const streamInitConcurrency = 4

type Platform struct {
	Name     string
	Channels []string
	Client   client.Client
}

type workerSet struct {
	cancel context.CancelFunc
	done   chan struct{}
}

func buildPlatforms(
	logger *zap.Logger,
	cfg *config.Config,
	initializeClients bool,
) []Platform {
	platforms := make([]Platform, 0, 4)
	if cfg.Twitch.Enable {
		platform := Platform{
			Name:     "twitch",
			Channels: cfg.Twitch.Channels,
		}

		if initializeClients {
			platform.Client = twitch.NewClient(
				logger,
				cfg.Twitch.Proxy.HTTP,
				cfg.Twitch.Proxy.Socks,
			)
		}

		platforms = append(
			platforms,
			platform,
		)
	}

	if cfg.Kick.Enable {
		platform := Platform{
			Name:     "kick",
			Channels: cfg.Kick.Channels,
		}

		if initializeClients {
			platform.Client = kick.NewClient(
				logger,
				cfg.Kick.Proxy.HTTP,
				cfg.Kick.Proxy.Socks,
			)
		}

		platforms = append(
			platforms,
			platform,
		)
	}

	if cfg.Youtube.Enable {
		platform := Platform{
			Name:     "youtube",
			Channels: cfg.Youtube.Channels,
		}

		if initializeClients {
			platform.Client = youtube.NewClient(
				logger,
				cfg.Youtube.Proxy.HTTP,
				cfg.Youtube.Proxy.Socks,
			)
		}

		platforms = append(
			platforms,
			platform,
		)
	}

	if cfg.WTV.Enable {
		platform := Platform{
			Name:     "wtv",
			Channels: cfg.WTV.Channels,
		}

		if initializeClients {
			platform.Client = wtv.NewClient(
				logger,
				cfg.WTV.Proxy.HTTP,
				cfg.WTV.Proxy.Socks,
			)
		}

		platforms = append(
			platforms,
			platform,
		)
	}

	return platforms
}

func buildApplications(
	icon string,
	logger *zap.Logger,
	platforms []Platform,
	checkInterval time.Duration,
	notificationService notifier.Notifier,
	stateStorage *storage.StateStorage,
) []*app.App {
	applications := make(
		[]*app.App,
		0,
		len(platforms),
	)

	for _, platform := range platforms {
		application := app.NewApp(
			icon,
			logger,
			platform.Name,
			platform.Channels,
			checkInterval,
			platform.Client,
			notificationService,
			stateStorage,
		)

		applications = append(applications, application)
	}

	return applications
}

func runApplications(
	ctx context.Context,
	logger *zap.Logger,
	applications []*app.App,
	platforms []Platform,
) error {
	group, ctx := errgroup.WithContext(ctx)
	for index, application := range applications {
		application := application
		platform := platforms[index]
		group.Go(
			func() error {
				if err := application.Run(ctx); err != nil {
					logger.Error(
						"Application stopped",
						zap.String(
							"Platform",
							platform.Name,
						),
						zap.Error(err),
					)
					return err
				}
				return nil
			},
		)
	}
	return group.Wait()
}

func startWorkers(
	parentCtx context.Context,
	icon string,
	logger *zap.Logger,
	cfg *config.Config,
	platforms []Platform,
	applications []*app.App,
	summaryInterval time.Duration,
	stateStorage *storage.StateStorage,
	notificationService notifier.Notifier,
) *workerSet {
	workerCtx, cancel := context.WithCancel(parentCtx)
	workers := &workerSet{
		cancel: cancel,
		done:   make(chan struct{}),
	}

	go func() {
		defer close(workers.done)

		var group sync.WaitGroup
		group.Add(1)

		go func() {
			defer group.Done()

			if err := runApplications(
				workerCtx,
				logger,
				applications,
				platforms,
			); err != nil {
				if workerCtx.Err() != nil {
					return
				}

				logger.Error(
					"Applications stopped with error",
					zap.Error(err),
				)
			}
		}()

		if cfg.Summary.Enable {
			group.Add(1)
			go func() {
				defer group.Done()
				runSummaryWorker(
					workerCtx,
					logger,
					summaryInterval,
					stateStorage,
					notificationService,
					icon,
				)
			}()
		}
		group.Wait()
	}()

	return workers
}

func runConfigWatcher(
	ctx context.Context,
	logger *zap.Logger,
	configPath string,
	configChanges <-chan struct{},
	configErrors <-chan error,
	configReloads chan<- *config.Config,
) error {
	for {
		select {
		case <-ctx.Done():
			return nil

		case _, ok := <-configChanges:
			if !ok {
				configChanges = nil
				if configErrors == nil {
					return nil
				}

				continue
			}

			newConfig, err := config.LoadConfig(configPath)
			if err != nil {
				logger.Error(
					"Reload config",
					zap.Error(err),
				)
				continue
			}

			select {
			case configReloads <- newConfig:
			case <-ctx.Done():
				return nil
			}

		case err, ok := <-configErrors:
			if !ok {
				configErrors = nil
				if configChanges == nil {
					return nil
				}

				continue
			}

			logger.Error(
				"Config watcher",
				zap.Error(err),
			)
		}
	}
}

func runConfigReloads(
	ctx context.Context,
	logger *zap.Logger,
	configReloads <-chan *config.Config,
	reloader *ConfigReloader,
) error {
	for {
		select {
		case <-ctx.Done():
			return nil

		case newConfig := <-configReloads:
			logger.Info("Config hot reload started...")
			if err := reloader.Reload(newConfig); err != nil {
				logger.Error(
					"Config hot reload failed",
					zap.Error(err),
				)

				continue
			}

			logger.Info("Config hot reload complete!")
		}
	}
}

func (w *workerSet) Stop() {
	if w == nil {
		return
	}

	w.cancel()
	<-w.done
}

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
