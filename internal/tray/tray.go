package tray

import (
	_ "embed"
	"tway/internal/tui"

	"github.com/getlantern/systray"
	"go.uber.org/zap"
)

//go:embed tway.ico
var icon []byte

type Tray struct {
	log    *zap.Logger
	onExit func()
}

func NewTray(
	log *zap.Logger,
	onExit func(),
) *Tray {
	return &Tray{
		log:    log,
		onExit: onExit,
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
		"Show status",
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
