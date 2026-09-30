package tui

import (
	"fmt"
	"net/url"
	"strings"
	"time"
	"tway/internal/config"
	"tway/internal/i18n"

	"github.com/atotto/clipboard"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

const (
	setupViewWidth          = 70
	setupViewHeight         = 17
	platformSetupViewHeight = 18
	channelsSetupViewHeight = 12
	proxySetupViewHeight    = 14
	addChannelViewHeight    = 10
)

type Setup struct {
	Name     string
	Settings *config.Platform
}

func (u *TUI) ShowSetup(
	config *config.Config,
) (bool, error) {
	saved := false

	var showMainSetup func()
	showMainSetup = func() {
		texts := i18n.Get(config.Language)
		form := tview.NewForm()

		languageIndex := 0
		if config.Language == i18n.Russian {
			languageIndex = 1
		}

		form.AddDropDown(
			texts.Language,
			[]string{
				"English",
				"Русский",
			},
			languageIndex,
			func(
				_ string, index int,
			) {
				language := i18n.English
				if index == 1 {
					language = i18n.Russian
				}

				if config.Language == language {
					return
				}

				config.Language = language
				showMainSetup()
			},
		)

		form.AddInputField(
			texts.CheckInterval,
			config.Check,
			20,
			intervalInputAccept,
			func(value string) {
				config.Check = strings.TrimSpace(value)
			},
		)

		addCheckbox(
			form,
			texts.Summary,
			config.Summary.Enable,
			func(checked bool) {
				config.Summary.Enable = checked
			},
		)

		form.AddInputField(
			texts.SummaryInterval,
			config.Summary.Interval,
			20,
			intervalInputAccept,
			func(value string) {
				config.Summary.Interval = strings.TrimSpace(value)
			},
		)

		addCheckbox(
			form,
			texts.ShowStreamersAfterSave,
			config.UI.ShowStreamers,
			func(checked bool) {
				config.UI.ShowStreamers = checked
			},
		)

		form.AddButton(
			texts.Platforms,
			func() {
				u.showPlatformSetup(
					config,
					showMainSetup,
					0,
					texts,
				)
			},
		)

		form.AddButton(
			texts.Save,
			func() {
				if err := validateInterval(
					config.Check,
					10*time.Second,
					texts,
				); err != nil {
					showInternalError(
						u,
						texts.CheckInterval,
						err,
						showMainSetup,
						texts,
					)
					return
				}

				if config.Summary.Enable {
					if err := validateInterval(
						config.Summary.Interval,
						time.Second*10,
						texts,
					); err != nil {
						showInternalError(
							u,
							texts.SummaryInterval,
							err,
							showMainSetup,
							texts,
						)
						return
					}
				}

				saved = true
				u.application.Stop()
			},
		)

		form.AddButton(
			texts.Cancel,
			func() {
				u.application.Stop()
			},
		)

		form.SetBorder(true).
			SetTitle(
				fmt.Sprintf(
					" %s ",
					texts.SetupTitle,
				),
			).
			SetTitleAlign(tview.AlignCenter)

		u.application.SetRoot(
			centerPrimitive(
				form,
				setupViewWidth,
				setupViewHeight,
			),
			true,
		)

		form.SetInputCapture(
			func(event *tcell.EventKey) *tcell.EventKey {
				switch event.Key() {
				case tcell.KeyEscape:
					u.application.Stop()
					return nil
				}

				switch event.Rune() {
				case 'q', 'Q', 'й', 'Й':
					u.application.Stop()
					return nil
				}

				return event
			},
		)
	}

	showMainSetup()
	if err := u.application.EnableMouse(true).Run(); err != nil {
		return false, err
	}

	return saved, nil
}

func (u *TUI) showPlatformSetup(
	config *config.Config,
	showMainSetup func(),
	activePlatform int,
	texts i18n.Texts,
) {
	platforms := setupPlatforms(config)
	if activePlatform < 0 || activePlatform >= len(platforms) {
		activePlatform = 0
	}

	platform := platforms[activePlatform]
	form := tview.NewForm()

	names := make(
		[]string,
		0,
		len(platforms),
	)

	for _, platform := range platforms {
		names = append(names, platform.Name)
	}

	form.AddDropDown(
		texts.Platform,
		names,
		activePlatform,
		func(
			_ string, index int,
		) {
			if index == activePlatform {
				return
			}

			u.showPlatformSetup(
				config,
				showMainSetup,
				index,
				texts,
			)
		},
	)

	addCheckbox(
		form,
		texts.Enable,
		platform.Settings.Enable,
		func(checked bool) {
			platform.Settings.Enable = checked
		},
	)

	form.AddButton(
		fmt.Sprintf(
			"%s (%d)",
			texts.Channels,
			len(platform.Settings.Channels),
		),
		func() {
			currentPlatform := activePlatform
			u.showChannelsSetup(
				platform.Name,
				platform.Settings,
				func() {
					u.showPlatformSetup(
						config,
						showMainSetup,
						currentPlatform,
						texts,
					)
				},
				texts,
			)
		},
	)

	form.AddButton(
		proxyButtonLabel(
			platform.Settings.Proxy,
			texts,
		),
		func() {
			currentPlatform := activePlatform
			u.showProxySetup(
				platform.Name,
				&platform.Settings.Proxy,
				func() {
					u.showPlatformSetup(
						config,
						showMainSetup,
						currentPlatform,
						texts,
					)
				},
				texts,
			)
		},
	)

	form.AddButton(
		texts.Back,
		func() {
			showMainSetup()
		},
	)

	form.SetBorder(true).
		SetTitle(
			fmt.Sprintf(
				" %s ",
				texts.Platforms,
			),
		).
		SetTitleAlign(tview.AlignCenter)

	u.application.SetRoot(
		centerPrimitive(
			form,
			setupViewWidth,
			platformSetupViewHeight,
		),
		true,
	)
}

func (u *TUI) showChannelsSetup(
	platformName string,
	platform *config.Platform,
	back func(),
	texts i18n.Texts,
) {
	u.showChannelsSetupAt(
		platformName,
		platform,
		back,
		0,
		texts,
	)
}

func (u *TUI) showChannelsSetupAt(
	platformName string,
	platform *config.Platform,
	back func(),
	selectedChannel int,
	texts i18n.Texts,
) {
	form := tview.NewForm()
	if len(platform.Channels) > 0 {
		if selectedChannel < 0 {
			selectedChannel = 0
		}

		if selectedChannel >= len(platform.Channels) {
			selectedChannel = len(platform.Channels) - 1
		}

		form.AddDropDown(
			texts.Channel,
			platform.Channels,
			selectedChannel,
			func(
				_ string,
				index int,
			) {
				selectedChannel = index
			},
		)

	} else {
		form.AddTextView(
			texts.Channel,
			texts.NoChannelsAdded,
			40,
			1,
			false,
			false,
		)
	}

	form.AddButton(
		texts.AddChannel,
		func() {
			u.showAddChannelSetup(
				platformName,
				platform,
				func() {
					selected := len(platform.Channels) - 1
					u.showChannelsSetupAt(
						platformName,
						platform,
						back,
						selected,
						texts,
					)
				},
				texts,
			)
		},
	)

	form.AddButton(
		texts.RemoveChannel,
		func() {
			if len(platform.Channels) == 0 {
				return
			}

			platform.Channels = append(
				platform.Channels[:selectedChannel],
				platform.Channels[selectedChannel+1:]...,
			)

			if selectedChannel >= len(platform.Channels) {
				selectedChannel = len(platform.Channels) - 1
			}

			if selectedChannel < 0 {
				selectedChannel = 0
			}

			u.showChannelsSetupAt(
				platformName,
				platform,
				back,
				selectedChannel,
				texts,
			)
		},
	)

	form.AddButton(
		texts.Back,
		back,
	)

	form.SetBorder(true).
		SetTitle(
			fmt.Sprintf(
				" %s ",
				fmt.Sprintf(
					texts.PlatformChannels,
					platformName,
				),
			),
		).
		SetTitleAlign(tview.AlignCenter)

	u.application.SetRoot(
		centerPrimitive(
			form,
			setupViewWidth,
			channelsSetupViewHeight,
		),
		true,
	)
}

func (u *TUI) showAddChannelSetup(
	platformName string,
	platform *config.Platform,
	back func(),
	texts i18n.Texts,
) {
	form := tview.NewForm()
	channel := ""

	channelInput := tview.NewInputField()

	channelInput.SetLabel(texts.Channel)
	channelInput.SetFieldWidth(50)
	channelInput.SetChangedFunc(
		func(value string) {
			channel = strings.TrimSpace(value)
		},
	)

	channelInput.SetInputCapture(
		func(event *tcell.EventKey) *tcell.EventKey {
			if event.Key() != tcell.KeyCtrlV {
				return event
			}

			value, err := clipboard.ReadAll()
			if err != nil {
				return nil
			}

			value = normalizeChannelInput(platformName, value)
			channelInput.SetText(value)
			channel = value

			return nil
		},
	)

	form.AddFormItem(channelInput)
	form.AddButton(
		texts.Add,
		func() {
			channel = normalizeChannelInput(platformName, channel)
			if channel == "" {
				return
			}

			if channelExists(
				platform.Channels,
				channel,
			) {
				return
			}

			platform.Channels = append(
				platform.Channels,
				channel,
			)

			back()
		},
	)

	form.AddButton(
		texts.Cancel,
		back,
	)

	form.SetBorder(true).
		SetTitle(
			fmt.Sprintf(
				" %s ",
				fmt.Sprintf(
					texts.AddPlatformChannel,
					platformName,
				),
			),
		).
		SetTitleAlign(tview.AlignCenter)

	u.application.SetRoot(
		centerPrimitive(
			form,
			setupViewWidth,
			addChannelViewHeight,
		),
		true,
	)
}

func (u *TUI) showProxySetup(
	platformName string,
	proxy *config.Proxy,
	back func(),
	texts i18n.Texts,
) {
	form := tview.NewForm()

	httpProxyInput := tview.NewInputField()
	httpProxyInput.SetLabel(texts.HTTPProxy)
	httpProxyInput.SetFieldWidth(50)
	httpProxyInput.SetText(proxy.HTTP)

	httpProxyInput.SetChangedFunc(
		func(value string) {
			proxy.HTTP = strings.TrimSpace(value)
		},
	)

	httpProxyInput.SetInputCapture(
		func(event *tcell.EventKey) *tcell.EventKey {
			if event.Key() != tcell.KeyCtrlV {
				return event
			}

			value, err := clipboard.ReadAll()
			if err != nil {
				return nil
			}

			value = strings.TrimSpace(value)
			httpProxyInput.SetText(value)

			return nil
		},
	)

	form.AddFormItem(httpProxyInput)

	socksProxyInput := tview.NewInputField()
	socksProxyInput.SetLabel(texts.SocksProxy)
	socksProxyInput.SetFieldWidth(50)
	socksProxyInput.SetText(proxy.Socks)

	socksProxyInput.SetChangedFunc(
		func(value string) {
			proxy.Socks = strings.TrimSpace(value)
		},
	)

	socksProxyInput.SetInputCapture(
		func(event *tcell.EventKey) *tcell.EventKey {
			if event.Key() != tcell.KeyCtrlV {
				return event
			}

			value, err := clipboard.ReadAll()
			if err != nil {
				return nil
			}

			value = strings.TrimSpace(value)
			socksProxyInput.SetText(value)

			return nil
		},
	)

	form.AddFormItem(socksProxyInput)

	form.AddButton(
		texts.ClearProxy,
		func() {
			proxy.HTTP = ""
			proxy.Socks = ""

			u.showProxySetup(
				platformName,
				proxy,
				back,
				texts,
			)
		},
	)

	form.AddButton(
		texts.Back,
		back,
	)

	form.SetBorder(true).
		SetTitle(
			fmt.Sprintf(
				" %s ",
				fmt.Sprintf(
					texts.PlatformProxy,
					platformName,
				),
			),
		).
		SetTitleAlign(tview.AlignCenter)

	u.application.SetRoot(
		centerPrimitive(
			form,
			setupViewWidth,
			proxySetupViewHeight,
		),
		true,
	)
}

func addCheckbox(
	form *tview.Form,
	label string,
	checked bool,
	changed func(bool),
) {
	checkbox := tview.NewCheckbox().
		SetLabel(label).
		SetChecked(checked).
		SetCheckedString(tview.Escape("[x]")).
		SetUncheckedString(tview.Escape("[ ]")).
		SetChangedFunc(changed)
	form.AddFormItem(checkbox)
}

func normalizeChannelInput(
	platformName, value string,
) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}

	rawURL := value
	if !strings.Contains(rawURL, "://") {
		switch {
		case strings.Contains(rawURL, "twitch.tv/"),
			strings.Contains(rawURL, "kick.com/"),
			strings.Contains(rawURL, "youtube.com/"),
			strings.Contains(rawURL, "youtu.be/"),
			strings.Contains(rawURL, "w.tv/"):
			rawURL = "https://" + rawURL

		default:
			return value
		}
	}

	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		return value
	}

	host := strings.ToLower(
		strings.TrimPrefix(
			parsedURL.Hostname(),
			"www.",
		),
	)

	path := strings.Trim(parsedURL.Path, "/")
	if path == "" {
		return value
	}

	parts := strings.Split(path, "/")
	switch strings.ToLower(platformName) {
	case "twitch":
		if host != "twitch.tv" {
			return value
		}

		return parts[0]

	case "kick":
		if host != "kick.com" {
			return value
		}

		return parts[0]

	case "youtube":
		if host != "youtube.com" &&
			host != "m.youtube.com" {
			return value
		}

		if strings.HasPrefix(parts[0], "@") {
			return parts[0]
		}

		if len(parts) >= 2 {
			switch parts[0] {
			case "channel", "c", "user":
				return parts[1]
			}
		}

		return value

	case "w.tv":
		if host != "w.tv" {
			return value
		}

		return parts[0]
	}

	return value
}

func validateInterval(
	value string,
	min time.Duration,
	texts i18n.Texts,
) error {
	value = strings.TrimSpace(value)
	if value == "" {
		return fmt.Errorf("%s", texts.IntervalCannotBeEmpty)
	}

	interval, err := time.ParseDuration(value)
	if err != nil {
		return fmt.Errorf("%s", texts.InvalidInterval)
	}

	if interval <= 0 {
		return fmt.Errorf("%s", texts.IntervalMustBeGreaterThanZero)
	}

	if interval < min {
		return fmt.Errorf(texts.IntervalMustBeAtLeast, min)
	}

	return nil
}

func intervalInputAccept(
	text string,
	lastChar rune,
) bool {
	if lastChar == 0 {
		return true
	}

	switch {
	case lastChar >= '0' && lastChar <= '9':
		return true

	case lastChar == '.':
		return true

	case lastChar == 'h',
		lastChar == 'm',
		lastChar == 's':
		return true
	}

	return false
}

func showInternalError(
	u *TUI,
	name string,
	err error,
	back func(),
	texts i18n.Texts,
) {
	modal := tview.NewModal().
		SetText(
			fmt.Sprintf(
				"%s:\n\n%s",
				name,
				err,
			),
		).
		AddButtons([]string{texts.OK}).
		SetDoneFunc(
			func(buttonIndex int, buttonLabel string) {
				back()
			},
		)

	u.application.SetRoot(modal, true)
}

func setupPlatforms(
	config *config.Config,
) []Setup {
	return []Setup{
		{
			Name:     "Twitch",
			Settings: &config.Twitch.Platform,
		},
		{
			Name:     "Kick",
			Settings: &config.Kick.Platform,
		},
		{
			Name:     "YouTube",
			Settings: &config.Youtube.Platform,
		},
		{
			Name:     "W.TV",
			Settings: &config.WTV.Platform,
		},
	}
}

func proxyButtonLabel(
	proxy config.Proxy,
	texts i18n.Texts,
) string {
	if strings.TrimSpace(proxy.HTTP) == "" &&
		strings.TrimSpace(proxy.Socks) == "" {
		return texts.AddProxy
	}

	return texts.EditProxy
}

func channelExists(
	channels []string,
	channel string,
) bool {
	for _, existing := range channels {
		if strings.EqualFold(existing, channel) {
			return true
		}
	}

	return false
}
