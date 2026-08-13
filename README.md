# Gofer

> Консольный мессенджер на Go: групповые каналы и личные сообщения в реальном времени, всё внутри терминала.

![Russian](https://img.shields.io/badge/lang-Russian-1793d1) [![English](https://img.shields.io/badge/lang-English-lightgrey)](README.en.md)

![status](https://img.shields.io/badge/status-alpha-orange) ![Go](https://img.shields.io/badge/Go-1.26-00ADD8) ![Postgres](https://img.shields.io/badge/PostgreSQL-16-336791) ![WebSocket](https://img.shields.io/badge/WebSocket-realtime-4A5568)

<!-- О ПРОЕКТЕ -->
<h2 align="left">💬 О проекте</h2>

<table>
<tr>
<td valign="top" width="55%">

**Gofer** — это клиент-серверный мессенджер, который целиком живёт в терминале.

- **Язык:** [**`Go 1.26`**](https://go.dev/)
- **База данных:** [**`PostgreSQL 16`**](https://www.postgresql.org/) в Docker
- **Реальное время:** [**`WebSocket`**](https://github.com/gorilla/websocket) — сообщения приходят мгновенно
- **Авторизация:** [**`JWT`**](https://github.com/golang-jwt/jwt) — access + refresh токены
- **Интерфейс:** [**`Bubble Tea`**](https://github.com/charmbracelet/bubbletea) — TUI с мышью и клавиатурой
- **Архитектура:** Clean Architecture, 4 слоя

Два бинарника: **сервер** (HTTP + WebSocket + Postgres) и **TUI-клиент**. Клиент умеет держать список серверов и переключаться между ними прямо из интерфейса.

</td>
<td valign="top" width="45%">

<img src="docs/screenshots/chat.png" alt="chat screenshot" />

</td>
</tr>
</table>

<!-- ГАЛЕРЕЯ -->
<a name="Gallery"></a>
<h2 align="center">🖼️ Галерея</h2>

<table>
<tr>
<td align="center" width="50%">
  <img src="docs/screenshots/register.png" alt="login and register"><br>
  <b>Вход / Регистрация</b>
</td>
<td align="center" width="50%">
  <img src="docs/screenshots/channels.png" alt="channels"><br>
  <b>Каналы</b>
</td>
</tr>
<tr>
<td align="center" width="50%">
  <img src="docs/screenshots/servers.png" alt="server picker"><br>
  <b>Выбор сервера</b>
</td>
<td align="center" width="50%">
  <img src="docs/screenshots/popup.png" alt="confirm popup"><br>
  <b>Подтверждение действия</b>
</td>
</tr>
</table>

<!-- ОГЛАВЛЕНИЕ -->
<a name="Table-of-Contents"></a>

## 📚 Оглавление

+ [Возможности](#Features)
+ [Требования](#Requirements)
+ [Установка и запуск](#Installation)
  + [Сценарий A — всё на одной машине](#Local)
  + [Сценарий B — сервер на удалённой машине](#Remote)
  + [Сборка бинарников](#Build)
+ [Конфигурация](#Configuration)
+ [Технологии](#Stack)

---

<!-- ВОЗМОЖНОСТИ -->
<a name="Features"></a>
<h2 align="center">🚀 Возможности</h2>

- **Авторизация по JWT** — регистрация и вход, access-токен (15 минут) + refresh-токен (7 дней). При обрыве WebSocket-соединения токен обновляется автоматически.
- **Каналы (групповые чаты)** — создать, войти по ID, выйти, удалить (только создатель), история сообщений.
- **Личные сообщения (DM)** — начать диалог по ID пользователя, удалить, история. Создание и удаление DM прилетают собеседнику мгновенно через WebSocket.
- **Сообщения в реальном времени** — доставка через WebSocket с подтверждением записи (ack) и защитой от дублей при повторной отправке.
- **Поиск пользователей** — по части имени, чтобы начать с кем-то диалог.
- **Выбор сервера в клиенте** — список серверов хранится локally, переключение и проверка доступности (`●` онлайн / офлайн) прямо из TUI, без пересборки.
- **Терминальный интерфейс** — навигация мышью и клавиатурой, индикатор соединения, копирование ID в буфер обмена, цветные имена в чате.
- **Graceful shutdown** — сервер аккуратно завершает соединения по сигналу.

[⬆ Наверх](#Table-of-Contents)

---

<!-- ТРЕБОВАНИЯ -->
<a name="Requirements"></a>
<h2 align="center">🧰 Требования</h2>

Для запуска понадобятся:

- **Go 1.26+** — [установка](https://go.dev/doc/install)
- **Docker** (с Docker Compose) — для PostgreSQL
- **golang-migrate CLI** — утилита для миграций базы. Ставится отдельно, в зависимостях проекта её нет:

```bash
go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest
```

> 💡 Бинарник `migrate` должен оказаться в `PATH` (обычно `~/go/bin`). Проверить: `migrate -version`.

- **Клиент буфера обмена** (опционально, только Linux) — для копирования ID: `xclip`, `xsel` или `wl-copy`. Без них всё работает, кроме копирования.

[⬆ Наверх](#Table-of-Contents)

---

<!-- УСТАНОВКА -->
<a name="Installation"></a>
<h2 align="center">⚡ Установка и запуск</h2>

Склонируйте репозиторий:

```bash
# HTTPS
git clone https://github.com/MrTrigraf/Gofer.git

# или SSH
git clone git@github.com:MrTrigraf/Gofer.git

cd Gofer
```

<a name="Local"></a>
### Сценарий A — всё на одной машине

Самый простой путь: сервер, база и клиент на одном компьютере. Все команды используют дефолты из `Makefile`.

**1. Поднять PostgreSQL в Docker:**

```bash
make docker
```

Поднимется контейнер `gofer_db` (PostgreSQL 16) на порту `5432`.

**2. Создать конфиг сервера:**

```bash
cp config/config.yaml.example config/config.yaml
```

Откройте `config/config.yaml` и **обязательно поменяйте JWT-секреты** (`jwt.secret` и `jwt.refresh`) на свои случайные строки. Остальные значения по умолчанию совпадают с `docker-compose.yml` и менять их не нужно.

**3. Применить миграции базы:**

```bash
make migrate-up
```

Создаст таблицы: пользователи, каналы, участники, личные чаты, сообщения.

**4. Запустить сервер:**

```bash
make run-server
```

Сервер поднимется на `http://localhost:8080`.

**5. В другом терминале запустить клиент:**

```bash
make run-client
```

Откроется TUI. При первом запуске список серверов пуст — нажмите кнопку **`[ SERVERS ]`** внизу экрана, добавьте адрес `http://localhost:8080`, выберите его, и можно регистрироваться.

<a name="Remote"></a>
### Сценарий B — сервер на удалённой машине

Сервер разворачивается на любой машине (VPS, домашний сервер), а клиенты подключаются к нему по сети.

**На сервере:**

1. Установите Go, поднимите PostgreSQL (через `make docker` или отдельно).
2. Соберите бинарник сервера (см. [Сборка](#Build)) или запустите через `go run ./cmd/server`.
3. Сервер читает конфиг по относительному пути `config/config.yaml`. Если бинарник лежит в другом месте — укажите путь явно флагом:

```bash
./gofer-server --config /etc/gofer/config.yaml
```

4. Примените миграции, указав адрес удалённой базы в строке подключения (см. `Makefile`, таргет `migrate-up`).
5. Откройте порт сервера (по умолчанию `8080`) наружу. Для продакшена рекомендуется поставить перед сервером reverse-proxy (nginx/Caddy) с TLS — клиент поддерживает `https`/`wss`.

**На клиенте:**

1. Запустите клиент (`make run-client` или собранный бинарник).
2. Нажмите **`[ SERVERS ]`** → **Add** → введите адрес сервера:
   - `http://your-server.com:8080` — обычное соединение;
   - `https://your-server.com` — если сервер за TLS-прокси (WebSocket сам переключится на `wss`).
3. Выберите сервер в списке (`●` покажет, доступен ли он) и подключайтесь.

> 💡 Адрес сервера больше **не** захардкожен — клиент хранит список серверов в `~/.config/gofer/servers.json` и позволяет переключаться между ними прямо из интерфейса.

<a name="Build"></a>
### Сборка бинарников

Собрать оба бинарника под текущую систему:

```bash
go build -o gofer-server ./cmd/server
go build -o gofer-client ./cmd/client
```

Кросс-компиляция под другую ОС/архитектуру (Go умеет это из коробки):

```bash
# Linux (amd64)
GOOS=linux   GOARCH=amd64 go build -o gofer-server-linux   ./cmd/server

# Windows
GOOS=windows GOARCH=amd64 go build -o gofer-client.exe     ./cmd/client

# macOS (Apple Silicon)
GOOS=darwin  GOARCH=arm64 go build -o gofer-client-macos   ./cmd/client
```

> 💡 Рядом с бинарником сервера должна быть папка `config/` с `config.yaml`, либо путь к конфигу задаётся флагом `--config`.

[⬆ Наверх](#Table-of-Contents)

---

<!-- КОНФИГУРАЦИЯ -->
<a name="Configuration"></a>
<h2 align="center">⚙️ Конфигурация</h2>

Сервер настраивается через `config/config.yaml`. Шаблон — `config/config.yaml.example`.

```yaml
server:
  port: "8080"              # порт HTTP + WebSocket

database:
  host: "localhost"
  port: "5432"
  user: "gofer"
  password: "gofer"
  name: "gofer"
  sslmode: "disable"
  max_conns: 25             # размер пула соединений
  min_conns: 5
  max_conn_lifetime: "5m"
  max_conn_idle_time: "2m"
  health_check_period: "1m"

jwt:
  secret: "your-secret-key"   # ⚠ поменяйте на свою случайную строку
  refresh: "your-refresh-key" # ⚠ поменяйте на свою случайную строку
  access_ttl: "15m"           # время жизни access-токена
  refresh_ttl: "168h"         # время жизни refresh-токена (7 дней)
```

> ⚠️ **Никогда не коммитьте `config/config.yaml` с реальными секретами.** В репозитории должен лежать только `.example`.

Клиент отдельного конфига не требует — список серверов хранится в `~/.config/gofer/servers.json` и наполняется через интерфейс.

[⬆ Наверх](#Table-of-Contents)

---

<!-- ТЕХНОЛОГИИ -->
<a name="Stack"></a>
<h2 align="center">📦 Технологии</h2>

<details>
<summary><b>Сервер</b></summary>

- [`jackc/pgx/v5`](https://github.com/jackc/pgx) — драйвер и пул соединений PostgreSQL
- [`gorilla/websocket`](https://github.com/gorilla/websocket) — WebSocket
- [`golang-jwt/jwt/v5`](https://github.com/golang-jwt/jwt) — JWT-токены
- [`golang.org/x/crypto/bcrypt`](https://pkg.go.dev/golang.org/x/crypto/bcrypt) — хеширование паролей
- [`goccy/go-yaml`](https://github.com/goccy/go-yaml) — чтение конфига
- [`google/uuid`](https://github.com/google/uuid) — UUID
- `net/http` — стандартный роутер (Go 1.22+)
- `log/slog` — структурированное логирование

</details>

<details>
<summary><b>Клиент (TUI)</b></summary>

- [`charmbracelet/bubbletea`](https://github.com/charmbracelet/bubbletea) — фреймворк TUI (архитектура Elm)
- [`charmbracelet/lipgloss`](https://github.com/charmbracelet/lipgloss) — стили и вёрстка терминала
- [`charmbracelet/bubbles`](https://github.com/charmbracelet/bubbles) — готовые компоненты (поля ввода, вьюпорт)
- [`gorilla/websocket`](https://github.com/gorilla/websocket) — WebSocket-клиент
- [`atotto/clipboard`](https://github.com/atotto/clipboard) — работа с буфером обмена

</details>

<details>
<summary><b>Инфраструктура</b></summary>

- [`PostgreSQL 16`](https://www.postgresql.org/) — база данных
- [`Docker Compose`](https://docs.docker.com/compose/) — запуск базы
- [`golang-migrate`](https://github.com/golang-migrate/migrate) — миграции схемы

</details>

[⬆ Наверх](#Table-of-Contents)
