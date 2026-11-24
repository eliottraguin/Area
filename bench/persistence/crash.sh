#!/usr/bin/env bash
# Scenario 4: kill the engine during writes, restart it, verify.
# `docker kill` sends SIGKILL to the process: this tests a process crash,
# not a power cut (the OS disk cache survives).
set -euo pipefail
cd "$(dirname "$0")"
ACK=$(mktemp)

run_container() { # <store> <container>
  go run . crash-writer "$1" "$ACK" & local wp=$!
  sleep 4
  docker kill "$2" >/dev/null
  wait $wp || true
  docker start "$2" >/dev/null
  until [ "$(docker inspect -f '{{.State.Health.Status}}' "$2")" = healthy ]; do sleep 1; done
  go run . crash-verify "$1" "$ACK"
}

docker compose up -d --wait
run_container postgres bench-postgres
run_container mongo bench-mongo
run_container mongo-w1 bench-mongo

for s in sqlite; do
  go run . crash-writer "$s" "$ACK" & wp=$!
  sleep 4
  pkill -KILL -f "bench.*crash-writer $s" || kill -9 $wp || true
  wait $wp 2>/dev/null || true
  go run . crash-verify "$s" "$ACK"
done
