#!/usr/bin/env bash
# Запуск Gofer: Postgres -> миграции -> сервер.
set -euo pipefail

DEPLOY_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$DEPLOY_DIR"

[[ -d ../migration ]] || { echo "ERROR: нет ../migration — deploy/ лежит не внутри клона Gofer" >&2; exit 1; }
[[ -f .env ]] || { echo "ERROR: нет .env — сначала ./setup.sh" >&2; exit 1; }
[[ -f config.prod.yaml ]] || { echo "ERROR: нет config.prod.yaml — сначала ./setup.sh" >&2; exit 1; }

COMPOSE=(docker compose -f docker-compose.prod.yml --env-file .env)

# --rebuild — пересобрать образ (после git pull)
if [[ "${1:-}" == "--rebuild" ]]; then
  echo "==> пересборка образа"
  "${COMPOSE[@]}" build
fi

echo "==> up"
"${COMPOSE[@]}" up -d

echo "==> статус"
"${COMPOSE[@]}" ps

# shellcheck disable=SC1091
source .env
IP="$(hostname -I 2>/dev/null | awk '{print $1}')"
echo
echo "Сервер: http://${IP:-<server-ip>}:${SERVER_PORT}"
echo "Логи:   docker compose -f docker-compose.prod.yml --env-file .env logs -f gofer_server"
