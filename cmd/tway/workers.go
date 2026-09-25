package main

import (
	"context"
	"sync"
	"time"
	"tway/internal/app"
	"tway/internal/config"
	"tway/internal/notifier"
	"tway/internal/storage"

	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
)

type workerSet struct {
	cancel context.CancelFunc
	done   chan struct{}
}

func createApplications(
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
				runSummaryLoop(
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

func (w *workerSet) stopAndWait() {
	if w == nil {
		return
	}

	w.cancel()
	<-w.done
}
