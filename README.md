# Tway

English | [Русский](README.ru.md)

Tway is a cross-platform stream notifier.

It checks your favorite streamers and sends a desktop notification when someone goes live.

Supported platforms:

- Twitch
- Kick
- YouTube
- W.TV

## Setup

Keep these files in the same folder:

```text
tway
tway.yaml
tway.ico
```

On Windows the executable will be:

```text
tway.exe
```

## Configuration

Tway reads its settings from `tway.yaml`.

Example:

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

### Check interval

```yaml
check: "2m"
```

How often Tway checks channels for status changes.

Examples:

```yaml
check: "30s"
check: "2m"
check: "1h"
```

### Summary

```yaml
summary:
  enable: true
  interval: "10m"
```

`enable` controls stream summary notifications:

```yaml
enable: true
```

enables them.

```yaml
enable: false
```

disables them.

`interval` controls how often the summary is shown.

### Adding streamers

Add the channel name under the platform's `channels` section:

```yaml
twitch:
  channels:
    - "forsen"
    - "xqc"
    - "shroud"
```

Use the channel name/handle from its URL.

For example:

```text
https://www.twitch.tv/forsen
                      ^^^^^^
```

Use:

```yaml
- "forsen"
```

You can add as many channels as you want.

Restart Tway after changing the configuration.

### Proxy

Each platform can use its own proxy:

```yaml
proxy:
  http: ""
  socks: ""
```

Leave both fields empty to connect directly:

```yaml
proxy:
  http: ""
  socks: ""
```

HTTP proxy example:

```yaml
proxy:
  http: "127.0.0.1:10808"
  socks: ""
```

SOCKS proxy example:

```yaml
proxy:
  http: ""
  socks: "127.0.0.1:10808"
```

## Run

### Windows

```bash
tway.exe
```

### Linux

```bash
./tway
```

Tway will start in the background and appear in the system tray.

## TUI

Tway also includes a terminal interface for viewing saved stream statuses.

Windows:

```bash
tway.exe --tui
```

Linux:

```bash
./tway --tui
```

## Custom paths

Use another config file:

```bash
tway.exe --config ./my-config.yaml
```

Use another icon:

```bash
tway.exe --icon ./my-icon.ico
```

Both options can be combined:

```bash
tway.exe --config ./my-config.yaml --icon ./my-icon.ico
```

## Build

Tway requires Go.

Build for your current platform:

```bash
go build -o app/tway ./cmd/tway
```

Windows release build:

```bash
go build -ldflags="-H windowsgui" -o app/tway.exe ./cmd/tway
```

After building, put `tway.yaml` and `tway.ico` next to the executable.

## License

Apache License 2.0