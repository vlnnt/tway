package tui

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type refreshSettings struct {
	interval time.Duration
	enabled  bool
}

func RunLogs(
	path string,
) error {
	app := tview.NewApplication()
	app.EnableMouse(true)

	logs := tview.NewTextView()
	logs.SetBorder(true)
	logs.SetTitle(" Tway Logs ")
	logs.SetScrollable(true)
	logs.SetWrap(false)
	logs.SetDynamicColors(false)

	intervalButton := tview.NewButton("Interval")
	pauseButton := tview.NewButton("Pause")
	clearButton := tview.NewButton("Clear")
	refreshButton := tview.NewButton("Refresh")
	closeButton := tview.NewButton("Close")

	status := tview.NewTextView()
	status.SetTextAlign(tview.AlignCenter)

	controls := tview.NewFlex().
		AddItem(status, 18, 0, false).
		AddItem(nil, 1, 0, false).
		AddItem(intervalButton, 12, 0, false).
		AddItem(nil, 1, 0, false).
		AddItem(pauseButton, 10, 0, false).
		AddItem(nil, 1, 0, false).
		AddItem(clearButton, 9, 0, false).
		AddItem(nil, 1, 0, false).
		AddItem(refreshButton, 11, 0, false).
		AddItem(nil, 1, 0, false).
		AddItem(closeButton, 9, 0, false)

	footer := tview.NewFlex().
		AddItem(nil, 0, 9, false).
		AddItem(controls, 74, 0, false).
		AddItem(nil, 0, 11, false)

	layout := tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(
			logs,
			0,
			1,
			true,
		).
		AddItem(
			footer,
			1,
			0,
			false,
		)

	refreshInterval := time.Second
	autoRefresh := true

	updateFooter := func() {
		state := "ON"
		pauseLabel := "Pause"

		if !autoRefresh {
			state = "OFF"
			pauseLabel = "Resume"
		}

		status.SetText(
			fmt.Sprintf(
				" Auto: %s (%s)",
				state,
				refreshInterval,
			),
		)

		pauseButton.SetLabel(pauseLabel)
	}

	load := func() {
		data, err := os.ReadFile(path)
		if err != nil {
			logs.SetText(
				fmt.Sprintf(
					"Failed to read log file: \n\n%s",
					err,
				),
			)
			return
		}

		logs.SetText(string(data))
		logs.ScrollToEnd()
	}

	clear := func() error {
		if err := os.Truncate(
			path,
			0,
		); err != nil {
			return fmt.Errorf(
				"clear log file: %w",
				err,
			)
		}

		load()
		return nil
	}

	refreshChanges := make(chan refreshSettings, 1)
	updateRefresh := func() {
		settings := refreshSettings{
			interval: refreshInterval,
			enabled:  autoRefresh,
		}

		select {
		case refreshChanges <- settings:
		default:
			select {
			case <-refreshChanges:
			default:
			}

			refreshChanges <- settings
		}

		updateFooter()
	}

	showClearConfirmation := func() {
		modal := tview.NewModal().
			SetText("Clear all logs?").
			AddButtons([]string{
				"Clear",
				"Cancel",
			}).
			SetDoneFunc(
				func(buttonIndex int, buttonLabel string) {
					if buttonLabel == "Clear" {
						if err := clear(); err != nil {
							errorModal := tview.NewModal().
								SetText(
									fmt.Sprintf(
										"Failed to clear logs:\n\n%s",
										err,
									),
								).
								AddButtons(
									[]string{
										"OK",
									},
								).
								SetDoneFunc(
									func(buttonIndex int, buttonLabel string) {
										app.SetRoot(
											layout,
											true,
										)
										app.SetFocus(logs)
									},
								)

							app.SetRoot(
								errorModal,
								true,
							)
							return
						}
					}

					app.SetRoot(
						layout,
						true,
					)
					app.SetFocus(logs)
				},
			)

		app.SetRoot(
			modal,
			true,
		)
	}

	showIntervalInput := func() {
		input := tview.NewInputField()
		input.SetLabel("Interval (seconds): ")
		input.SetText(
			fmt.Sprintf(
				"%g",
				refreshInterval.Seconds(),
			),
		)
		input.SetFieldWidth(10)

		errorText := tview.NewTextView()
		errorText.SetTextAlign(tview.AlignCenter)

		form := tview.NewForm()
		form.SetBorder(true)
		form.SetTitle(" Auto Refresh ")
		form.AddFormItem(input)

		closeForm := func() {
			app.SetRoot(
				layout,
				true,
			)

			app.SetFocus(logs)
		}

		form.AddButton(
			"Save",
			func() {
				value := strings.TrimSpace(input.GetText())
				interval, err := time.ParseDuration(value + "s")
				if err != nil || interval <= 0 {
					errorText.SetText("Enter a number greater than 0")
					return
				}

				refreshInterval = interval
				updateRefresh()
				closeForm()
			},
		)

		form.AddButton(
			"Cancel",
			closeForm,
		)

		form.SetInputCapture(
			func(event *tcell.EventKey) *tcell.EventKey {
				if event.Key() == tcell.KeyEscape {
					closeForm()
					return nil
				}

				return event
			},
		)

		dialog := tview.NewFlex().
			SetDirection(tview.FlexRow).
			AddItem(nil, 0, 1, false).
			AddItem(tview.NewFlex().
				AddItem(nil, 0, 1, false).
				AddItem(tview.NewFlex().
					SetDirection(tview.FlexRow).
					AddItem(form, 7, 0, true).
					AddItem(errorText, 1, 0, false),
					45,
					0,
					true,
				).
				AddItem(nil, 0, 1, false),
				8,
				0,
				true,
			).
			AddItem(nil, 0, 1, false)

		app.SetRoot(
			dialog,
			true,
		)

		app.SetFocus(input)
	}

	intervalButton.SetSelectedFunc(
		func() {
			showIntervalInput()
		},
	)

	pauseButton.SetSelectedFunc(
		func() {
			autoRefresh = !autoRefresh
			updateRefresh()
		},
	)

	clearButton.SetSelectedFunc(
		func() {
			showClearConfirmation()
		},
	)

	refreshButton.SetSelectedFunc(
		func() {
			load()
		},
	)

	closeButton.SetSelectedFunc(
		func() {
			app.Stop()
		},
	)

	logs.SetInputCapture(
		func(event *tcell.EventKey) *tcell.EventKey {
			switch event.Key() {
			case tcell.KeyEscape:
				app.Stop()
				return nil

			case tcell.KeyF5:
				load()
				return nil
			}

			switch event.Rune() {
			case 'q', 'Q':
				app.Stop()
				return nil

			case 'r', 'R':
				load()
				return nil

			case 'c', 'C':
				showClearConfirmation()
				return nil

			case 'i', 'I':
				showIntervalInput()
				return nil

			case ' ':
				autoRefresh = !autoRefresh
				updateRefresh()
				return nil
			}

			return event
		},
	)

	load()
	updateFooter()

	stopRefresh := make(chan struct{})
	go func() {
		interval := refreshInterval
		enabled := autoRefresh

		timer := time.NewTimer(interval)
		defer timer.Stop()

		for {
			var timerChannel <-chan time.Time
			if enabled {
				timerChannel = timer.C
			}

			select {
			case <-timerChannel:
				app.QueueUpdateDraw(
					func() {
						load()
					},
				)
				timer.Reset(interval)

			case settings := <-refreshChanges:
				if !timer.Stop() {
					select {
					case <-timer.C:
					default:
					}
				}

				interval = settings.interval
				enabled = settings.enabled

				if enabled {
					timer.Reset(interval)
				}

			case <-stopRefresh:
				return
			}
		}
	}()

	err := app.
		SetRoot(layout, true).
		SetFocus(logs).
		Run()

	close(stopRefresh)
	if err != nil {
		return fmt.Errorf(
			"run logs TUI: %w",
			err,
		)
	}

	return nil
}
