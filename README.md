# Tway

Cross-platform stream notifier with desktop notifications.

Currently supporting platforms - Kick, Twitch, Youtube, w.tv

## Config

Config example:

```json
{
  "check": "2m",
  "summary": {
      "enable": true,
      "interval": "10m"
  },
  "twitch": {
      "http_proxy": "127.0.0.1:10808",
      "socks_proxy": "",
      "channels": [
          "forsen",
      ]
  },
  "kick": {
      "http_proxy": "",
      "socks_proxy": "127.0.0.1:10808",
      "channels": [
        "forsen",
      ]
  },
  "youtube": {
      "http_proxy": "127.0.0.1:10808",
      "socks_proxy": "",
      "channels": [
        "forsen",
      ]
  },
  "wtv": {
      "http_proxy": "",
      "socks_proxy": "",
      "channels": [
        "forsen"
      ]
  }
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

## License

Apache License 2.0