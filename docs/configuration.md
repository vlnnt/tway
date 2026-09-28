# Configuration

Tway stores its settings in `tway.yaml`.

The file is created after the initial setup and can be edited manually or through the built-in settings interface.

Configuration changes are applied automatically while Tway is running.

## Example

```yaml
language: "en"

check: "2m"

summary:
  enable: true
  interval: "10m"

ui:
  show_streamers: true

logs:
  interval: "10s"

twitch:
  enable: true
  proxy:
    http: ""
    socks: ""
  channels:
    - "forsen"

kick:
  enable: true
  proxy:
    http: ""
    socks: ""
  channels:
    - "forsen"

youtube:
  enable: true
  proxy:
    http: "127.0.0.1:10808"
    socks: ""
  channels:
    - "forsen"

wtv:
  enable: true
  proxy:
    http: ""
    socks: ""
  channels:
    - "forsen"
```

## Language

```yaml
language: "en"
```

Controls the application language.

Supported values:

```text
en    English
ru    Russian
```

Example:

```yaml
language: "ru"
```

## Check interval

```yaml
check: "2m"
```

Controls how often Tway checks stream status.

Examples:

```yaml
check: "30s"
check: "2m"
check: "1h"
```

Supported duration suffixes include:

```text
s    seconds
m    minutes
h    hours
```

The minimum check interval is `10s`.

## Summary

```yaml
summary:
  enable: true
  interval: "10m"
```

Controls periodic stream summary notifications.

### `enable`

```yaml
enable: true
```

Enables summary notifications.

```yaml
enable: false
```

Disables them.

### `interval`

```yaml
interval: "10m"
```

Controls how often Tway sends a stream summary.

The minimum summary interval is `10s`.

## UI

```yaml
ui:
  show_streamers: true
```

### `show_streamers`

Controls whether the stream status window is opened after saving the configuration.

```yaml
show_streamers: true
```

Enables it.

```yaml
show_streamers: false
```

Disables it.

## Logs

```yaml
logs:
  interval: "10s"
```

Controls the automatic refresh interval of the built-in log viewer.

Examples:

```yaml
interval: "1s"
interval: "10s"
interval: "1m"
```

The interval can also be changed directly from the log viewer.

## Platforms

Tway currently supports:

- Twitch
- Kick
- YouTube
- W.TV

Each platform has the same basic configuration structure:

```yaml
twitch:
  enable: true
  proxy:
    http: ""
    socks: ""
  channels:
    - "forsen"
```

### `enable`

Controls whether the platform is monitored.

```yaml
enable: true
```

Enables the platform.

```yaml
enable: false
```

Disables the platform.

## Channels

Channels are listed under `channels`:

```yaml
channels:
  - "forsen"
  - "xqc"
  - "shroud"
```

An empty channel list can be written as:

```yaml
channels: []
```

Use the channel name or handle from the streamer's URL.

Example:

```text
https://www.twitch.tv/forsen
                      ^^^^^^
```

Configuration:

```yaml
channels:
  - "forsen"
```

Channels can also be added or removed through the built-in settings interface.

## Proxy

Each platform can use its own proxy configuration:

```yaml
proxy:
  http: ""
  socks: ""
```

Leave both values empty to connect directly.

### HTTP proxy

```yaml
proxy:
  http: "127.0.0.1:10808"
  socks: ""
```

### SOCKS proxy

```yaml
proxy:
  http: ""
  socks: "127.0.0.1:10808"
```

Proxy settings are configured independently for each platform.

## Full platform example

```yaml
twitch:
  enable: true
  proxy:
    http: ""
    socks: ""
  channels:
    - "forsen"
    - "xqc"
```

To disable the platform without removing its channels:

```yaml
twitch:
  enable: false
  proxy:
    http: ""
    socks: ""
  channels:
    - "forsen"
    - "xqc"
```

The channels remain in the configuration and can be enabled again later.