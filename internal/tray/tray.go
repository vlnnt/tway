package tray

import (
	_ "embed"
	"tway/internal/app"
	"tway/internal/config"
	"tway/internal/notifier"
	"tway/internal/tui"
	"tway/internal/twitch"

	"github.com/getlantern/systray"
	"go.uber.org/zap"
)

//go:embed twitch.ico
var icon []byte

type Tray struct {
	log      *zap.Logger
	config   *config.Config
	twitch   *twitch.Client
	notifier notifier.Notifier
	storage  *app.StateStorage
	onExit   func()
}

func NewTray(
	log *zap.Logger,
	config *config.Config,
	twitch *twitch.Client,
	notifier notifier.Notifier,
	storage *app.StateStorage,
	onExit func(),
) *Tray {
	return &Tray{
		log:      log,
		config:   config,
		twitch:   twitch,
		notifier: notifier,
		storage:  storage,
		onExit:   onExit,
	}
}

func (t *Tray) Run() {
	systray.Run(t.onReady, t.onExitHandler)
}

func (t *Tray) showStreamers() {
	if err := tui.OpenTerminal(); err != nil {
		t.log.Error(
			"Tray.showStreamers.OpenTerminal",
			zap.Error(err),
		)
	}
}

func (t *Tray) onReady() {
	systray.SetTitle("tway")
	systray.SetTooltip("tway")
	systray.SetIcon(icon)

	show := systray.AddMenuItem(
		"Show Status",
		"Show streamers status",
	)

	systray.AddSeparator()
	exit := systray.AddMenuItem(
		"Exit",
		"Exit application",
	)

	go func() {
		for range show.ClickedCh {
			t.log.Info("Tray.showStreamers",
				zap.String("Clicked", "Tray show streamers status requested"),
			)

			t.showStreamers()
		}
	}()

	go func() {
		<-exit.ClickedCh
		t.log.Info("Tray.onReady",
			zap.String("Clicked", "Tray exit requested"))
		systray.Quit()
	}()
}

func (t *Tray) onExitHandler() {
	if t.onExit != nil {
		t.onExit()
	}
}
