package main

import (
	"context"
	"encoding/json"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"tway/internal/app"
	"tway/internal/config"
	"tway/internal/notifier"
	"tway/internal/tray"
	"tway/internal/twitch"
)

func main() {
	log.Println("Starting Tway...")
	exePath, err := os.Executable()
	if err != nil {
		log.Fatal(err)
	}

	configPath := filepath.Join(
		filepath.Dir(exePath),
		"stream.json",
	)

	if len(os.Args) >= 2 {
		configPath = os.Args[1]
	}

	log.Printf("Loading config from %q...", configPath)
	file, err := os.Open(configPath)
	if err != nil {
		log.Fatalf("Open config: %v", err)
	}
	defer file.Close()

	var config config.Config
	if err := json.NewDecoder(file).Decode(&config); err != nil {
		log.Fatalf("Decode config: %v", err)
	}

	log.Printf(
		"Config loaded: interval=%s, streamers=%d",
		config.CheckInterval.Duration,
		len(config.Streamers),
	)

	twitchClient := twitch.NewClient()
	log.Println("Twitch client initialized!")

	notificationService, err := notifier.New()
	if err != nil {
		log.Fatalf("Create notifier: %v", err)
	}
	defer notificationService.Close()

	log.Println("Notifier initialized!")
	storage := app.NewStateStorage(
		"state.json",
	)

	application := app.New(
		&config,
		twitchClient,
		notificationService,
		storage,
	)

	log.Println("Application initialized!")
	ctx, stop := signal.NotifyContext(
		context.Background(),
		os.Interrupt,
		syscall.SIGTERM,
	)
	defer stop()

	go func() {
		if err := application.Run(ctx); err != nil {
			log.Printf("Application stopped: %v", err)
			stop()
		}
	}()

	trayApp := tray.New(func() {
		log.Println("Tray exit requested!")
		stop()
	})

	log.Println("Tray started!")
	trayApp.Run()
	log.Println("Tway stopped!")
}
