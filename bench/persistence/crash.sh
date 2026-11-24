#!/usr/bin/env bash
# Scénario 4 : tuer le moteur pendant l'écriture, redémarrer, vérifier.
# `docker kill` envoie SIGKILL au processus : cela teste le crash du processus,
# pas la coupure de courant (le cache disque de l'OS survit).
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
