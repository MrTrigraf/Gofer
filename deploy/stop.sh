#!/usr/bin/env bash
# Остановка Gofer. Данные БД остаются в $DATA_DIR/postgres.
set -euo pipefail

DEPLOY_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$DEPLOY_DIR"

[[ -f .env ]] || { echo "ERROR: нет .env" >&2; exit 1; }

docker compose -f docker-compose.prod.yml --env-file .env down
echo "остановлено. Данные БД сохранены (bind-mount, docker down их не трогает)."
