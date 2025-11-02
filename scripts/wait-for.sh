#!/usr/bin/env bash
set -euo pipefail

host="${1:-localhost}"
port="${2:-80}"
shift 2 || true

until nc -z "$host" "$port"; do
  echo "waiting for $host:$port"
  sleep 1
done

if [ "$#" -gt 0 ]; then
  exec "$@"
fi
