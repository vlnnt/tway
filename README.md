# Tway

Cross-platform Twitch stream watcher with desktop notifications.

## Features

- Windows/Linux support
- System tray application
- Live/offline notifications
- Multiple streamer tracking
- Configurable check interval

## Config

Create `stream.json`:

```json
{
  "check_interval": "1m",
  "streamers": [
    {
      "channel": "streamer_name"
    }
  ]
}
```

## Run

Windows:

```bash
tway.exe
```

Linux:

```bash
./tway
```

## Build

```bash
go build -o app/tway ./cmd/tway
```

Windows release:

```bash
go build -ldflags="-H windowsgui" -o app/tway.exe ./cmd/tway
```

## Tech Stack

- Go
- Twitch GraphQL API
- Windows Toast Notifications
- Linux D-Bus Notifications
- System Tray

## License

Apache License 2.0