package tui

import (
	"fmt"
	"os/exec"
	"runtime"
	"time"

	"tway/internal/client"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type Loader func() ([]*client.Stream, error)

type TUI struct {
	application *tview.Application
}

func NewTUI() *TUI {
	return &TUI{
		application: tview.NewApplication(),
	}
}

func (u *TUI) ShowStreamers(
	load Loader,
) error {
	loading := tview.NewTextView().
		SetTextAlign(tview.AlignCenter).
		SetDynamicColors(true)

	loading.SetBorder(true).
		SetTitle(" Streams ").
		SetTitleAlign(tview.AlignCenter)

	u.application.SetRoot(
		centerPrimitive(loading, 50, 5),
		true,
	)

	go u.loadStreams(loading, load)
	return u.application.
		EnableMouse(true).
		Run()
}

func (u *TUI) loadStreams(
	loading *tview.TextView,
	load Loader,
) {
	frames := []string{
		"|",
		"/",
		"-",
		"\\",
	}

	done := make(chan struct{})
	go func() {
		ticker := time.NewTicker(80 * time.Millisecond)
		defer ticker.Stop()
		frame := 0

		for {
			select {
			case <-done:
				return

			case <-ticker.C:
				currentFrame := frames[frame%len(frames)]
				frame++

				u.application.QueueUpdateDraw(
					func() {
						loading.SetText(
							fmt.Sprintf(
								"\nLoading streams status %s",
								currentFrame,
							),
						)
					},
				)
			}
		}
	}()

	states, err := load()
	close(done)

	if err != nil {
		u.application.QueueUpdateDraw(
			func() {
				errorView := buildErrorView(err)
				u.application.SetRoot(
					centerPrimitive(
						errorView,
						70,
						7,
					),
					true,
				)
			},
		)

		return
	}

	table := buildTable(states)
	u.application.QueueUpdateDraw(
		func() {
			u.application.SetRoot(
				table,
				true,
			)
		},
	)
}

func buildTable(
	states []*client.Stream,
) *tview.Table {
	table := tview.NewTable().
		SetBorders(true).
		SetSelectable(false, false)

	table.SetTitle(" Streams ").
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
		if state == nil {
			continue
		}

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

		url := state.URL
		linkCell := tview.NewTableCell(url).
			SetAlign(tview.AlignCenter).
			SetTextColor(tcell.ColorLightSkyBlue).
			SetAttributes(tcell.AttrUnderline)

		linkCell.SetClickedFunc(
			func() bool {
				openURL(url)
				return true
			},
		)

		table.SetCell(
			row,
			2,
			linkCell,
		)

		row++
	}

	return table
}

func centerPrimitive(
	primitive tview.Primitive,
	width, height int,
) tview.Primitive {
	return tview.NewFlex().
		AddItem(nil, 0, 1, false).
		AddItem(
			tview.NewFlex().
				SetDirection(tview.FlexRow).
				AddItem(nil, 0, 1, false).
				AddItem(primitive, height, 1, true).
				AddItem(nil, 0, 1, false),
			width,
			1,
			true,
		).
		AddItem(nil, 0, 1, false)
}

func buildErrorView(
	err error,
) *tview.TextView {
	view := tview.NewTextView().
		SetTextAlign(tview.AlignCenter).
		SetDynamicColors(true)

	view.SetBorder(true).
		SetTitle(" Error ").
		SetTitleAlign(tview.AlignCenter)

	view.SetText(
		fmt.Sprintf(
			"\n[red]Failed to load streams status[-]\n\n%s",
			err.Error(),
		),
	)

	return view
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
