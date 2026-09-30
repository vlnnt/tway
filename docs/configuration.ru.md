# Конфигурация

Tway хранит настройки в файле `tway.yaml`.

Файл создаётся после первоначальной настройки и может редактироваться вручную или через встроенное меню настроек.

Изменения конфигурации применяются автоматически во время работы Tway.

## Пример

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

## Язык

```yaml
language: "en"
```

Определяет язык интерфейса приложения.

Поддерживаемые значения:

```text
en    Английский
ru    Русский
```

Пример:

```yaml
language: "ru"
```

## Интервал проверки

```yaml
check: "2m"
```

Определяет, как часто Tway проверяет состояние стримов.

Примеры:

```yaml
check: "30s"
check: "2m"
check: "1h"
```

Поддерживаемые обозначения времени:

```text
s    секунды
m    минуты
h    часы
```

Минимальный интервал проверки — `10s`.

## Сводка

```yaml
summary:
  enable: true
  interval: "10m"
```

Управляет периодическими уведомлениями со сводкой по стримам.

### `enable`

```yaml
enable: true
```

Включает уведомления со сводкой.

```yaml
enable: false
```

Выключает их.

### `interval`

```yaml
interval: "10m"
```

Определяет, как часто Tway отправляет сводку.

Минимальный интервал сводки — `10s`.

## Интерфейс

```yaml
ui:
  show_streamers: true
```

### `show_streamers`

Определяет, будет ли открываться окно состояния стримеров после сохранения конфигурации.

```yaml
show_streamers: true
```

Включает открытие окна.

```yaml
show_streamers: false
```

Выключает его.

## Логи

```yaml
logs:
  interval: "10s"
```

Определяет интервал автоматического обновления встроенного просмотрщика логов.

Примеры:

```yaml
interval: "1s"
interval: "10s"
interval: "1m"
```

Интервал также можно изменить прямо из окна логов.

## Платформы

Tway поддерживает:

- Twitch
- Kick
- YouTube
- W.TV

Для каждой платформы используется одинаковая базовая структура конфигурации:

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

Определяет, будет ли платформа отслеживаться.

```yaml
enable: true
```

Включает платформу.

```yaml
enable: false
```

Выключает платформу.

## Каналы

Каналы указываются в разделе `channels`:

```yaml
channels:
  - "forsen"
  - "xqc"
  - "shroud"
```

Пустой список каналов можно указать так:

```yaml
channels: []
```

Используйте имя канала или handle из ссылки на стримера.

Пример:

```text
https://www.twitch.tv/forsen
                      ^^^^^^
```

В конфигурации:

```yaml
channels:
  - "forsen"
```

Каналы также можно добавлять и удалять через встроенное меню настроек.

## Прокси

Для каждой платформы можно настроить отдельный прокси:

```yaml
proxy:
  http: ""
  socks: ""
```

Оставьте оба значения пустыми для прямого подключения без прокси.

### HTTP-прокси

```yaml
proxy:
  http: "127.0.0.1:10808"
  socks: ""
```

### SOCKS-прокси

```yaml
proxy:
  http: ""
  socks: "127.0.0.1:10808"
```

Настройки прокси задаются отдельно для каждой платформы.

## Полный пример платформы

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

Чтобы отключить платформу, не удаляя каналы:

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

Каналы останутся в конфигурации, и платформу можно будет включить снова позже.