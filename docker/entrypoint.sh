#!/bin/sh
set -e

/app/backend &
backend_pid=$!

trap 'kill $backend_pid 2>/dev/null' TERM INT EXIT

cd /app/frontend
exec node build
