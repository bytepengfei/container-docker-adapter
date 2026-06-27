#!/bin/sh
set -eu

if [ "${APPLE_CONTAINER_E2E:-}" != "1" ]; then
  echo "set APPLE_CONTAINER_E2E=1 to run the real Apple Container E2E test" >&2
  exit 2
fi

root=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
socket="${TMPDIR:-/tmp}/container-docker-adapter-e2e-$$.sock"
binary="${TMPDIR:-/tmp}/container-docker-adapter-e2e-$$"

cleanup() {
  if [ -n "${server_pid:-}" ]; then
    kill "$server_pid" 2>/dev/null || true
    wait "$server_pid" 2>/dev/null || true
  fi
  rm -f "$socket" "$binary"
}
trap cleanup EXIT INT TERM

container system status >/dev/null
GOCACHE="${GOCACHE:-$root/.gocache}" go build -o "$binary" "$root/cmd/dockerd-compat"
"$binary" -backend apple -container-bin /usr/local/bin/container -socket "$socket" &
server_pid=$!

i=0
while [ ! -S "$socket" ]; do
  i=$((i + 1))
  if [ "$i" -gt 100 ]; then
    echo "adapter socket was not created" >&2
    exit 1
  fi
  sleep 0.05
done

docker -H "unix://$socket" version >/dev/null
docker -H "unix://$socket" info >/dev/null
docker -H "unix://$socket" ps -a >/dev/null
docker -H "unix://$socket" images >/dev/null
output=$(docker -H "unix://$socket" run --rm hello-world)
printf '%s\n' "$output" | grep "Hello from Docker!" >/dev/null

remaining=$(docker -H "unix://$socket" ps -aq)
if [ -n "$remaining" ]; then
  echo "containers remain after docker run --rm: $remaining" >&2
  exit 1
fi

echo "Apple Container E2E passed"
