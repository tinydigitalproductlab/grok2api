#!/usr/bin/env bash
set -euo pipefail
DIR="$(cd "$(dirname "$0")" && pwd)"
cd "$DIR"
# Start only if not already listening on 8000
if curl -fsS --max-time 3 http://127.0.0.1:8000/healthz >/dev/null 2>&1; then
  echo "grok2api already running on 127.0.0.1:8000"
  exit 0
fi
nohup ./grok2api --config ./config.yaml --listen 127.0.0.1:8000 > grok2api.log 2>&1 < /dev/null &
echo "grok2api started pid $!"
