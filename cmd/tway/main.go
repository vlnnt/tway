package main

import (
	"context"
	"encoding/json"
	"fmt"
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

	if len(os.Args) < 2 {
		log.Fatal("usage: tway <config-path>")
	}

	configPath := os.Args[1]
	file, err := os.Open(configPath)
	if err != nil {
		log.Fatalf("open config %q: %v", configPath, err)
	}
	defer file.Close()

	var config config.Config
	if err := json.NewDecoder(file).Decode(&config); err != nil {
		log.Fatalf("decode config %q: %v", configPath, err)
	}

	twitchClient := twitch.NewClient()
	notificationService, err := notifier.New()
	if err != nil {
		log.Fatalf("create notifier: %v", err)
	}
	defer func() {
		if err := notificationService.Close(); err != nil {
			log.Printf("close notifier: %v", err)
		}
	}()

	application := app.NewApp(
		&config,
		twitchClient,
		notificationService,
	)

	ctx, stop := signal.NotifyContext(
		context.Background(),
		os.Interrupt,
		syscall.SIGTERM,
	)
	defer stop()

	if err := application.Run(ctx); err != nil {
		log.Fatal(fmt.Errorf("run app: %w", err))
	}
}
