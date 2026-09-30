# Tway

English | [Русский](README.ru.md)

Tway is a cross-platform desktop notifier for live streams.

It monitors your favorite streamers and sends desktop notifications when they go live or end a stream.

Supported platforms:

- Twitch
- Kick
- YouTube
- W.TV

## Run

Windows:

```bash
tway.exe
```

Linux:

```bash
./tway
```

Keep `tway.ico` next to the executable.

On the first launch, Tway opens the setup interface automatically.

## Commands

```text
--tui      Open stream status
--setup    Open settings
--logs     Open log viewer
--config   Use another config file
--icon     Use another icon
```

Example:

```bash
tway.exe --config ./my-config.yaml --icon ./my-icon.ico
```

## Configuration

See [Configuration](docs/configuration.md).

## Build

Linux:

```bash
./build.sh
```

Windows:

```bat
build.bat
```

## License

Apache License 2.0