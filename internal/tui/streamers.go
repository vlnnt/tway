package tui

import (
	"tway/internal/twitch"

	"github.com/rivo/tview"
)

type TUI struct {
	application *tview.Application
}

func NewTUI() *TUI {
	return &TUI{
		application: tview.NewApplication(),
	}
}

func (u *TUI) ShowStreamers(
	states []*twitch.Stream,
) error {
	table := tview.NewTable().
		SetBorders(true).
		SetSelectable(false, false)

	table.SetTitle(" Twitch Streamers ").
		SetBorder(true)

	table.SetCell(
		0,
		0,
		tview.NewTableCell("Streamer").
			SetAlign(tview.AlignCenter).
			SetExpansion(1),
	)

	table.SetCell(
		0,
		1,
		tview.NewTableCell("Status").
			SetAlign(tview.AlignCenter).
			SetExpansion(1),
	)

	row := 1
	for _, state := range states {
		status := "🔴 OFFLINE"
		if state.IsLive {
			status = "🟢 LIVE"
		}

		table.SetCell(
			row,
			0,
			tview.NewTableCell(state.Channel),
		)

		table.SetCell(
			row,
			1,
			tview.NewTableCell(status).
				SetAlign(tview.AlignCenter),
		)

		row++
	}

	return u.application.
		SetRoot(table, true).Run()
}
