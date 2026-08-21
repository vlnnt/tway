# Tway

Cross-platform stream notifier with desktop notifications.

Currently supporting platforms - Kick, Twitch, Youtube, w.tv

## Config

Config example:

```json
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

## License

Apache License 2.0