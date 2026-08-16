# Деплой Gofer

Развёртывание сервера Gofer на Linux-машине: PostgreSQL, миграции и HTTP/WebSocket-сервер в Docker Compose.

## Главное требование

**`deploy/` работает только внутри полного клона репозитория.** Копировать одну эту папку на сервер нельзя: образ собирается из корня репозитория (`context: ..` в `docker-compose.prod.yml`), а миграции монтируются из `../migration`.

Симптом, если скопировать только `deploy/`:

```
failed to solve: failed to compute cache key: "/go.sum" not found: not found
```

Скрипты проверяют это на старте и падают с понятным сообщением.

## Требования к серверу

| Требование | Значение |
|---|---|
| ОС | Linux x86_64 (проверено на Debian 12) |
| Docker | daemon запущен; `docker compose` v2 поставится автоматически, если его нет |
| Права | root (нужен для `chown` конфига и записи в `/srv`) |
| Пакеты | `git`, `curl`, `openssl` (`curl` доустановится сам) |
| RAM | 2 ГБ; при 1 ГБ сборка Go может быть убита OOM |
| Диск | ~2 ГБ на образы + данные БД |
| Порты | `SERVER_PORT` (по умолчанию 8080) свободен и открыт в firewall |

Go на сервере **не нужен** — сборка идёт внутри Docker.

## Установка

```bash
apt-get update && apt-get install -y git curl openssl
cd /opt
git clone https://github.com/MrTrigraf/Gofer.git
cd Gofer/deploy
./setup.sh
./start.sh
```

Клиент подключается по адресу `http://<IP-сервера>:8080` (добавляется в TUI).

### Переменные для `setup.sh`

Задаются только при первом запуске, дальше берутся из `.env`:

```bash
DATA_DIR=/srv/gofer-data SERVER_PORT=8080 ./setup.sh
```

- `DATA_DIR` — где лежат файлы БД (по умолчанию `/srv/gofer-data`, данные в `$DATA_DIR/postgres`)
- `SERVER_PORT` — внешний HTTP-порт (по умолчанию `8080`)

## Скрипты

### `setup.sh` — один раз при установке

1. Проверяет, что `deploy/` внутри клона репозитория
2. Проверяет Docker, ставит `docker compose` v2 плагином, если его нет
3. Создаёт `$DATA_DIR/postgres`
4. Генерирует пароль Postgres и два JWT-ключа → `.env` и `secrets.txt` (оба `chmod 600`)
5. Пишет `config.prod.yaml` для сервера, владелец — uid 10001 (пользователь внутри контейнера)
6. Собирает образ сервера

Повторный запуск безопасен: если `.env` уже есть, секреты не перегенерируются (иначе сломался бы доступ к существующей БД и разлогинились все пользователи).

### `start.sh` — запуск

Поднимает в порядке: Postgres → ожидание healthcheck → миграции → сервер.

```bash
./start.sh              # обычный запуск
./start.sh --rebuild    # пересобрать образ (после git pull)
```

### `stop.sh` — остановка

`docker compose down`. Данные БД остаются в `$DATA_DIR/postgres` — это bind-mount, `down` его не трогает.

## Обновление версии

```bash
cd /opt/Gofer
git pull
./deploy/start.sh --rebuild
```

Новые миграции накатятся автоматически при старте. Если миграций нет, контейнер `gofer_migrate` пишет `no change` и завершается — это норма.

## Что где лежит

| Файл | Назначение | В git |
|---|---|---|
| `Dockerfile` | multi-stage сборка сервера, финальный образ на alpine, запуск под non-root uid 10001 | да |
| `docker-compose.prod.yml` | три сервиса: `gofer_db`, `gofer_migrate`, `gofer_server` | да |
| `setup.sh` / `start.sh` / `stop.sh` | скрипты установки и управления | да |
| `.env` | переменные Compose, включая пароль БД и JWT-ключи | **нет** |
| `secrets.txt` | те же секреты в читаемом виде, для хранения в менеджере паролей | **нет** |
| `config.prod.yaml` | конфиг сервера, монтируется в `/etc/gofer/config.yaml` | **нет** |

Три последних файла генерирует `setup.sh`, и они в `.gitignore`.

## Диагностика

```bash
cd /opt/Gofer/deploy
COMPOSE="docker compose -f docker-compose.prod.yml --env-file .env"

$COMPOSE ps                      # статус контейнеров
$COMPOSE logs -f gofer_server    # логи сервера
$COMPOSE logs gofer_migrate      # результат миграций
curl http://localhost:8080/api/v1/health   # должно вернуть {"status":"ok"}
docker exec -it gofer_db psql -U gofer -d gofer   # консоль БД
```

Частые проблемы:

- **`"/go.sum" not found`** — скопирована только папка `deploy/`, нужен полный клон репозитория
- **`permission denied` на `/etc/gofer/config.yaml`** — `setup.sh` запускался не от root, `chown` не сработал; перезапустить от root
- **Сборка убита (`signal: killed`)** — не хватило RAM; добавить swap: `fallocate -l 2G /swapfile && chmod 600 /swapfile && mkswap /swapfile && swapon /swapfile`
- **Порт занят** — `ss -tlnp | grep 8080`, либо задать другой `SERVER_PORT` в `.env` и перезапустить

## Безопасность

- Пароль БД и JWT-ключи генерируются случайно при `setup.sh`. Файлы `.env`, `secrets.txt`, `config.prod.yaml` имеют права `600` и не попадают в git — не коммить их и не пересылать в открытых каналах.
- Порт Postgres наружу не публикуется: доступ к БД только из docker-сети.
- **Трафик идёт по HTTP без шифрования.** Пароли при регистрации и логине, а также JWT-токены передаются открытым текстом и могут быть перехвачены на любом промежуточном узле. Для реального использования нужен TLS — обратный прокси (Caddy/nginx) с сертификатом и домен.
- Смена JWT-ключей разлогинивает всех пользователей.
- Резервная копия БД: `docker exec gofer_db pg_dump -U gofer gofer > backup.sql`. Копирование `$DATA_DIR/postgres` на живой базе не даёт согласованного снимка — либо `pg_dump`, либо копировать после `stop.sh`.
