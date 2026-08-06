package tray

import (
	_ "embed"
	"log"

	"github.com/getlantern/systray"
)

//go:embed tray.ico
var icon []byte

type Tray struct {
	onExit func()
}

func New(
	onExit func(),
) *Tray {
	return &Tray{
		onExit: onExit,
	}
}

func (t *Tray) Run() {
	systray.Run(t.onReady, t.onExitHandler)
}

func (t *Tray) onReady() {
	systray.SetTitle("tway")
	systray.SetTooltip("Twitch watcher")
	systray.SetIcon(icon)

	status := systray.AddMenuItem(
		"Статус: работает",
		"Current status",
	)

	status.Disable()
	systray.AddSeparator()
	exit := systray.AddMenuItem(
		"Выход",
		"Exit application",
	)

	go func() {
		<-exit.ClickedCh
		log.Println("Tray exit requested")
		systray.Quit()
	}()
}

func (t *Tray) onExitHandler() {
	if t.onExit != nil {
		t.onExit()
	}
}
