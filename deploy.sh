#!/usr/bin/env bash
# Despliega el commit indicado. Se ejecuta en el servidor

set -euo pipefail # Strict mode

SHA="${1:?uso: ./deploy.sh <sha>}"

cd "$(dirname "$0")"

# Asserts SHA is the same as HEAD
if [[ "$(git rev-parse HEAD)" != "$SHA" ]]; then 
	echo "HEAD ($(git rev-parse --short HEAD)) no coincide con $SHA" >&2
	exit 1
fi

echo "Desplegando $SHA"

export ONYX_TAG="$SHA"
docker compose pull api
docker compose up -d --no-build
docker compose restart web # Restarts Caddy

echo "Deploy completado"
