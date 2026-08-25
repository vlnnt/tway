# Tway

[English](README.md) | Русский

Tway — утилита, которая следит за стримерами и присылает уведомление, когда кто-то начинает стрим.

Поддерживаются:

- Twitch
- Kick
- YouTube
- W.TV

## Как настроить

Все настройки находятся в `tway.yaml`.

Пример:

```yaml
check: "2m"

summary:
  enable: true
  interval: "10m"

twitch:
  proxy:
    http: ""
    socks: ""
  channels:
    - "forsen"

kick:
  proxy:
    http: ""
    socks: ""
  channels:
    - "forsen"

youtube:
  proxy:
    http: "127.0.0.1:10808"
    socks: ""
  channels:
    - "forsen"

wtv:
  proxy:
    http: ""
    socks: ""
  channels:
    - "forsen"
```

### Проверка каналов

```yaml
check: "2m"
```

Как часто Tway будет проверять стримеров.

Например:

```yaml
check: "30s" # каждые 30 секунд
check: "2m"  # каждые 2 минуты
check: "1h"  # каждый час
```

### Сводка

```yaml
summary:
  enable: true
  interval: "10m"
```

`enable` — включать сводку или нет.

```yaml
enable: true
```

Сводка включена.

```yaml
enable: false
```

Сводка выключена.

`interval` — как часто её присылать.

### Как добавить стримера

Добавьте его имя в `channels` нужной платформы.

Например:

```yaml
twitch:
  channels:
    - "forsen"
    - "xqc"
    - "shroud"
```

Имя берётся из ссылки на канал:

```text
https://www.twitch.tv/forsen
                      ^^^^^^
```

Значит в конфиг нужно добавить:

```yaml
- "forsen"
```

После изменения `tway.yaml` перезапустите программу.

### Прокси

Для каждой платформы можно отдельно указать HTTP или SOCKS-прокси.

Без прокси:

```yaml
proxy:
  http: ""
  socks: ""
```

HTTP:

```yaml
proxy:
  http: "127.0.0.1:10808"
  socks: ""
```

SOCKS:

```yaml
proxy:
  http: ""
  socks: "127.0.0.1:10808"
```

Если прокси не нужен — просто оставьте оба поля пустыми.

## Как запустить

Файлы `tway`, `tway.yaml` и `tway.ico` лучше держать в одной папке.

### Windows

```bash
tway.exe
```

### Linux

```bash
./tway
```

После запуска Tway появится в трее и начнёт проверять каналы из конфига.

## TUI

Состояние стримеров можно посмотреть прямо в терминале.

Windows:

```bash
tway.exe --tui
```

Linux:

```bash
./tway --tui
```

## Другой конфиг

Если конфиг лежит в другом месте:

```bash
tway.exe --config ./my-config.yaml
```

Для своей иконки:

```bash
tway.exe --icon ./my-icon.ico
```

Можно указать оба параметра сразу:

```bash
tway.exe --config ./my-config.yaml --icon ./my-icon.ico
```

## Как собрать

Нужен установленный Go.

Обычная сборка:

```bash
go build -o app/tway ./cmd/tway
```

Windows-версия без консольного окна:

```bash
go build -ldflags="-H windowsgui" -o app/tway.exe ./cmd/tway
```

После сборки положите рядом `tway.yaml` и `tway.ico`.

## Лицензия

Apache License 2.0