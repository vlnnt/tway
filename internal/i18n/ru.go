package i18n

var russian = Texts{
	Language: "Язык",

	SetupTitle:             "Настройки Tway",
	CheckInterval:          "Интервал проверки",
	Summary:                "Сводка",
	SummaryInterval:        "Интервал сводки",
	ShowStreamersAfterSave: "Показать окно стримов после сохранения",
	Platforms:              "Платформы",
	Platform:               "Платформа",
	Save:                   "Сохранить",
	Cancel:                 "Отмена",

	Enable:        "Включено",
	Channels:      "Каналы",
	Channel:       "Канал",
	Add:           "Добавить",
	AddChannel:    "Добавить канал",
	RemoveChannel: "Удалить канал",
	Back:          "Назад",
	OK:            "OK",

	NoChannelsAdded: "Каналы не добавлены",

	HTTPProxy:  "HTTP-прокси",
	SocksProxy: "SOCKS-прокси",
	ClearProxy: "Очистить прокси",
	AddProxy:   "Добавить прокси",
	EditProxy:  "Изменить прокси",

	PlatformChannels:   "Каналы %s",
	AddPlatformChannel: "Добавить канал %s",
	PlatformProxy:      "Прокси %s",

	IntervalCannotBeEmpty:         "Интервал не может быть пустым",
	InvalidInterval:               "Некорректный интервал: используйте значения вроде 30s, 2m или 1h",
	IntervalMustBeGreaterThanZero: "Интервал должен быть больше 0",
	IntervalMustBeAtLeast:         "Интервал должен быть не меньше %s",

	Streams:                  "Стримы",
	Streamer:                 "Стример",
	Status:                   "Статус",
	LastStream:               "Последний стрим",
	LiveFor:                  "В эфире",
	Live:                     "ОНЛАЙН",
	Offline:                  "НЕ В СЕТИ",
	Loading:                  "Загрузка статусов стримов",
	NoPlatforms:              "Нет включённых платформ.",
	EnablePlatformInSettings: "Включите платформу в настройках.",
	PlatformStreams:          "Стримы %s",
	StreamsStatusBar:         "В эфире: %d / %d | Tab/Shift+Tab: платформа | Q/Esc: выход",
	Error:                    "Ошибка",
	FailedToLoadStreams:      "Не удалось загрузить статусы стримов",
	MinuteShort:              "м",
	HourShort:                "ч",

	LogsTitle: "Логи Tway",

	Interval: "Интервал",
	Pause:    "Пауза",
	Resume:   "Продолжить",
	Clear:    "Очистить",
	Close:    "Закрыть",

	Auto: "Авто",
	On:   "ВКЛ",
	Off:  "ВЫКЛ",

	ClearAllLogs:               "Очистить все логи?",
	FailedToReadLogFile:        "Не удалось прочитать файл логов",
	FailedToClearLogs:          "Не удалось очистить логи",
	IntervalSeconds:            "Интервал (секунды): ",
	AutoRefresh:                "Автообновление",
	EnterNumberGreaterThanZero: "Введите число больше 0",

	Refresh:  "Обновить",
	Logs:     "Логи",
	Settings: "Настройки",
	Exit:     "Выход",
	Show:     "Показать",
	OpenLogs: "Открыть логи",

	ShowStreamersStatus:  "Показать статус стримеров",
	ShowStreamsSummary:   "Показать сводку стримов",
	RefreshStreamsStatus: "Обновить статус стримов",
	OpenTwaySettings:     "Открыть настройки Tway",
	OpenTwayLogFile:      "Открыть файл логов Tway",
	ExitApplication:      "Выйти из приложения",

	StreamStartedTitle:   "%s теперь в эфире!",
	StreamStartedMessage: "%s\nКатегория: %s",
	StreamEndedTitle:     "%s больше не в эфире!",
	StreamEndedMessage:   "Стример завершил трансляцию!",

	StreamManualRefreshAlreadyRunningMessage: "Ручное обновление статусов стримов уже выполняется!",
	StreamManualRefreshRequestMessage:        "Запущено обновление статусов стримов!",
	StreamManualRefreshStatusReadyMessage:    "Статусы стримов обновлены!",

	StreamOnlineMessage:  "Онлайн",
	StreamOfflineMessage: "Не в сети",

	StreamMonitoringClosedMessage: "Мониторинг стримов остановлен из-за критической ошибки. Tway будет закрыт.",

	StreamSingleInstanceMessage: "Tway уже запущен. Одновременно может работать только один экземпляр.",

	StreamInitializationMessage:  "Инициализация сервисов и подключение к стриминговым платформам...",
	StreamMonitoringReadyMessage: "Инициализация завершена. Все сервисы подключены, мониторинг стримов активен.",
}
