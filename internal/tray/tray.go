package tray

import (
	_ "embed"

	"github.com/getlantern/systray"
	"go.uber.org/zap"
)

//go:embed tray.ico
var icon []byte

type Tray struct {
	log    *zap.Logger
	onExit func()
}

func New(
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

func (t *Tray) onReady() {
	systray.SetTitle("tway")
	systray.SetTooltip("tway")
	systray.SetIcon(icon)

	systray.AddSeparator()
	exit := systray.AddMenuItem(
		"Exit",
		"Exit application",
	)

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
