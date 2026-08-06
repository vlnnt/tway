package main

import (
	"context"
	"encoding/json"
	"log"
	"os"
	"os/signal"
	"syscall"

	"tway/internal/app"
	"tway/internal/config"
	"tway/internal/notifier"
	"tway/internal/twitch"
)

func main() {
	log.Println("Starting Tway...")

	if len(os.Args) < 2 {
		log.Fatal("Usage: tway <config-path>")
	}

	configPath := os.Args[1]

	log.Printf("Loading config from %q...", configPath)

	file, err := os.Open(configPath)
	if err != nil {
		log.Fatalf("Open config: %v", err)
	}
	defer file.Close()

	var cfg config.Config

	if err := json.NewDecoder(file).Decode(&cfg); err != nil {
		log.Fatalf("Decode config: %v", err)
	}

	log.Printf(
		"Config loaded: interval=%s, streamers=%d",
		cfg.CheckInterval.Duration,
		len(cfg.Streamers),
	)

	twitchClient := twitch.NewClient()
	log.Println("Twitch client initialized")

	notificationService, err := notifier.New()
	if err != nil {
		log.Fatalf("Create notifier: %v", err)
	}
	defer notificationService.Close()

	log.Println("Notifier initialized")

	storage := app.NewStateStorage(
		"state.json",
	)

	application := app.New(
		&cfg,
		twitchClient,
		notificationService,
		storage,
	)

	log.Println("Application initialized")

	ctx, stop := signal.NotifyContext(
		context.Background(),
		os.Interrupt,
		syscall.SIGTERM,
	)
	defer stop()

	log.Println("Application started")

	if err := application.Run(ctx); err != nil {
		log.Fatalf("Application stopped with error: %v", err)
	}

	log.Println("Application stopped")
}
