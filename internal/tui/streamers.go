package tui

import (
	"os/exec"
	"runtime"

	"tway/internal/twitch"

	"github.com/gdamore/tcell/v2"
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
			SetExpansion(1).
			SetAttributes(tcell.AttrBold),
	)

	table.SetCell(
		0,
		1,
		tview.NewTableCell("Status").
			SetAlign(tview.AlignCenter).
			SetExpansion(1).
			SetAttributes(tcell.AttrBold),
	)

	table.SetCell(
		0,
		2,
		tview.NewTableCell("Link").
			SetAlign(tview.AlignCenter).
			SetExpansion(1).
			SetAttributes(tcell.AttrBold),
	)

	row := 1
	for _, state := range states {
		status := "OFFLINE"
		statusColor := tcell.ColorRed

		if state.IsLive {
			status = "LIVE"
			statusColor = tcell.ColorGreen
		}

		table.SetCell(
			row,
			0,
			tview.NewTableCell(state.Channel).
				SetAlign(tview.AlignCenter),
		)

		table.SetCell(
			row,
			1,
			tview.NewTableCell(status).
				SetAlign(tview.AlignCenter).
				SetTextColor(statusColor).
				SetAttributes(tcell.AttrBold),
		)

		url := "https://twitch.tv/" + state.Channel

		linkCell := tview.NewTableCell(url).
			SetAlign(tview.AlignCenter).
			SetTextColor(tcell.ColorLightSkyBlue).
			SetAttributes(tcell.AttrUnderline)

		linkCell.SetClickedFunc(func() bool {
			openURL(url)
			return true
		})

		table.SetCell(
			row,
			2,
			linkCell,
		)

		row++
	}

	return u.application.
		SetRoot(table, true).
		EnableMouse(true).
		Run()
}

func openURL(
	url string,
) {
	switch runtime.GOOS {
	case "windows":
		_ = exec.Command(
			"rundll32",
			"url.dll,FileProtocolHandler",
			url,
		).Start()

	case "linux":
		_ = exec.Command(
			"xdg-open",
			url,
		).Start()
	}
}
