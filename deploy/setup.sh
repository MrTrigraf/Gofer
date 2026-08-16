#!/usr/bin/env bash
# Первичная настройка Gofer на сервере: docker compose v2, секреты, каталог данных.
# Запускать один раз из каталога deploy/: sudo ./setup.sh
set -euo pipefail

DEPLOY_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$DEPLOY_DIR"

# --- 0. deploy/ должен лежать внутри клона репозитория -----------------------
# Образ собирается из корня репозитория (context: ..), миграции монтируются
# из ../migration. Одной скопированной папки deploy/ недостаточно.
for required in go.mod go.sum migration cmd/server; do
  if [[ ! -e "../$required" ]]; then
    printf '\033[1;31mERROR:\033[0m рядом нет ../%s — deploy/ лежит не внутри клона Gofer.\n' "$required" >&2
    printf 'Нужен полный репозиторий:\n  git clone https://github.com/MrTrigraf/Gofer.git\n  cd Gofer/deploy && ./setup.sh\n' >&2
    exit 1
  fi
done

DATA_DIR="${DATA_DIR:-/srv/gofer-data}"
SERVER_PORT="${SERVER_PORT:-8080}"
COMPOSE_V2_VERSION="v2.29.7"

log() { printf '\033[1;32m==>\033[0m %s\n' "$*"; }
die() { printf '\033[1;31mERROR:\033[0m %s\n' "$*" >&2; exit 1; }

# --- 1. docker ---------------------------------------------------------------
command -v docker >/dev/null || die "docker не установлен"
docker info >/dev/null 2>&1 || die "docker daemon не запущен (systemctl start docker)"

# --- 2. docker compose v2 (плагин) ------------------------------------------
if docker compose version >/dev/null 2>&1; then
  log "docker compose v2 уже есть: $(docker compose version --short)"
else
  log "ставлю docker compose $COMPOSE_V2_VERSION как CLI-плагин"
  command -v curl >/dev/null || { apt-get update && apt-get install -y curl; }
  PLUGIN_DIR=/usr/local/lib/docker/cli-plugins
  mkdir -p "$PLUGIN_DIR"
  ARCH="$(uname -m)"
  curl -fsSL \
    "https://github.com/docker/compose/releases/download/${COMPOSE_V2_VERSION}/docker-compose-linux-${ARCH}" \
    -o "$PLUGIN_DIR/docker-compose"
  chmod +x "$PLUGIN_DIR/docker-compose"
  docker compose version >/dev/null || die "docker compose так и не заработал"
  log "docker compose установлен"
fi

# --- 3. каталог данных БД ----------------------------------------------------
log "каталог данных: $DATA_DIR/postgres"
mkdir -p "$DATA_DIR/postgres"

# --- 4. секреты --------------------------------------------------------------
gen() { openssl rand -base64 48 | tr -d '/+=\n' | cut -c1-"${1:-32}"; }

if [[ -f .env ]]; then
  log ".env уже существует — секреты не трогаю"
else
  log "генерирую секреты"
  POSTGRES_USER="gofer"
  POSTGRES_DB="gofer"
  POSTGRES_PASSWORD="$(gen 32)"
  JWT_SECRET="$(gen 64)"
  JWT_REFRESH="$(gen 64)"

  umask 077
  cat > .env <<EOF
# Сгенерировано setup.sh $(date -Is). НЕ КОММИТИТЬ.
POSTGRES_USER=${POSTGRES_USER}
POSTGRES_PASSWORD=${POSTGRES_PASSWORD}
POSTGRES_DB=${POSTGRES_DB}
DATA_DIR=${DATA_DIR}
SERVER_PORT=${SERVER_PORT}
JWT_SECRET=${JWT_SECRET}
JWT_REFRESH=${JWT_REFRESH}
EOF
  chmod 600 .env

  cat > secrets.txt <<EOF
Gofer — секреты сервера (сгенерировано $(date -Is))
=========================================================
PostgreSQL
  host (внутри docker): gofer_db:5432
  database: ${POSTGRES_DB}
  user:     ${POSTGRES_USER}
  password: ${POSTGRES_PASSWORD}

JWT
  secret:  ${JWT_SECRET}
  refresh: ${JWT_REFRESH}

Каталог данных БД: ${DATA_DIR}/postgres
HTTP порт сервера:  ${SERVER_PORT}
=========================================================
Хранить вне репозитория. Смена JWT-ключей разлогинит всех.
EOF
  chmod 600 secrets.txt
  log "секреты записаны в deploy/.env и deploy/secrets.txt (chmod 600)"
fi

# shellcheck disable=SC1091
set -a; source .env; set +a

# --- 5. config.prod.yaml для сервера ----------------------------------------
log "пишу config.prod.yaml"
umask 077
cat > config.prod.yaml <<EOF
# Сгенерировано setup.sh. НЕ КОММИТИТЬ.
server:
  port: "8080"

database:
  host: "gofer_db"
  port: "5432"
  user: "${POSTGRES_USER}"
  password: "${POSTGRES_PASSWORD}"
  name: "${POSTGRES_DB}"
  sslmode: "disable"
  max_conns: 25
  min_conns: 5
  max_conn_lifetime: "5m"
  max_conn_idle_time: "2m"
  health_check_period: "1m"

jwt:
  secret: "${JWT_SECRET}"
  refresh: "${JWT_REFRESH}"
  access_ttl: "15m"
  refresh_ttl: "168h"
EOF
chmod 600 config.prod.yaml
# Контейнер сервера работает под uid 10001 (см. Dockerfile), поэтому файл
# должен принадлежать этому uid — иначе внутри контейнера "permission denied".
if chown 10001:10001 config.prod.yaml 2>/dev/null; then
  log "config.prod.yaml: 600, владелец uid 10001 (пользователь контейнера)"
else
  chmod 644 config.prod.yaml
  printf '\033[1;33mWARNING:\033[0m chown не удался (нужен root). config.prod.yaml выставлен в 644 — секреты читаемы всем пользователям хоста. Запусти setup.sh от root.\n' >&2
fi

# --- 6. сборка образа --------------------------------------------------------
log "собираю образ сервера (первый раз долго: 1 CPU / 2 ГБ RAM)"
docker compose -f docker-compose.prod.yml --env-file .env build

log "готово. Запуск: ./start.sh   Остановка: ./stop.sh"
