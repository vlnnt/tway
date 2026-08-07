package tray

import (
	_ "embed"
	"fmt"
	"strings"
	"tway/internal/config"
	"tway/internal/notifier"
	"tway/internal/twitch"

	"github.com/getlantern/systray"
	"go.uber.org/zap"
)

//go:embed twitch.ico
var icon []byte

type Tray struct {
	icon     string
	log      *zap.Logger
	config   *config.Config
	twitch   *twitch.Client
	notifier notifier.Notifier
	onExit   func()
}

func New(
	icon string,
	log *zap.Logger,
	config *config.Config,
	twitch *twitch.Client,
	notifier notifier.Notifier,
	onExit func(),
) *Tray {
	return &Tray{
		icon:     icon,
		log:      log,
		config:   config,
		twitch:   twitch,
		notifier: notifier,
		onExit:   onExit,
	}
}

func (t *Tray) Run() {
	systray.Run(t.onReady, t.onExitHandler)
}

func (t *Tray) checkStreamers() {
	var status strings.Builder
	status.WriteString("Twitch streamers status:\n\n")
	for _, streamer := range t.config.Streamers {
		channel := streamer.Channel
		stream, err := t.twitch.GetStream(channel)
		if err != nil {
			t.log.Error("Tray.checkStreamers.GetStream", zap.Error(err))
			continue
		}

		if stream.IsLive {
			status.WriteString(fmt.Sprintf("🟢 %s — LIVE\n", channel))
		} else {
			status.WriteString(fmt.Sprintf("🔴 %s — OFFLINE\n", channel))
		}
	}

	if err := t.notifier.Send(notifier.Notification{
		Title:   "tway",
		Message: status.String(),
		Icon:    t.icon,
	}); err != nil {
		t.log.Error("Tray.checkStreamers.Send", zap.Error(err))
	}
}

func (t *Tray) onReady() {
	systray.SetTitle("tway")
	systray.SetTooltip("tway")
	systray.SetIcon(icon)

	check := systray.AddMenuItem(
		"Refresh status",
		"Refresh streamers status",
	)

	systray.AddSeparator()
	exit := systray.AddMenuItem(
		"Exit",
		"Exit application",
	)

	go func() {
		for range check.ClickedCh {
			t.log.Info("Tray.checkStreamers",
				zap.String("Clicked", "Tray check streamers requested"))
			t.checkStreamers()
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
