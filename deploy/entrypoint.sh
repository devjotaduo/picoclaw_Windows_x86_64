#!/bin/sh
# Launches the PicoClaw web launcher in production.
# - Seeds /data/config.json from the template on first boot (the /data volume
#   persists the password, credentials and config across redeploys).
# - Binds on 0.0.0.0:$PORT (Railway/Fly inject $PORT; defaults to 18800).
set -e

DATA_DIR="${PICOCLAW_DATA:-/data}"
CONFIG="$DATA_DIR/config.json"
PORT="${PORT:-18800}"

mkdir -p "$DATA_DIR"
if [ ! -f "$CONFIG" ]; then
  echo "seeding $CONFIG from template"
  cp /app/config.template.json "$CONFIG"
fi

exec picoclaw web -public -addr "0.0.0.0:$PORT" -config "$CONFIG"
