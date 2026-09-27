#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

if ! command -v podman >/dev/null 2>&1; then
  echo "podman is required" >&2
  exit 1
fi

if [[ "$(podman info --format '{{.Host.Security.Rootless}}')" != "true" ]]; then
  echo "refusing to run: this test requires rootless Podman" >&2
  exit 1
fi

if [[ "${LPIC_DAILY_SKIP_IMAGE_BUILD:-0}" != "1" ]]; then
  podman build     --tag localhost/lpic-daily/fedora-phase1:1     --file labs/images/fedora-phase1/Containerfile     .
fi

runtime_dir="${XDG_RUNTIME_DIR:-${TMPDIR:-/tmp}/lpic-daily-runtime-$(id -u)}"
mkdir -p "$runtime_dir/podman"
chmod 0700 "$runtime_dir"
export XDG_RUNTIME_DIR="$runtime_dir"

socket="$XDG_RUNTIME_DIR/podman/podman.sock"
rm -f "$socket"

podman system service --time=0 "unix://$socket" >"$XDG_RUNTIME_DIR/podman/service.log" 2>&1 &
service_pid=$!
cleanup() {
  kill "$service_pid" 2>/dev/null || true
  wait "$service_pid" 2>/dev/null || true
}
trap cleanup EXIT

for _ in $(seq 1 50); do
  if [[ -S "$socket" ]]; then
    break
  fi
  if ! kill -0 "$service_pid" 2>/dev/null; then
    cat "$XDG_RUNTIME_DIR/podman/service.log" >&2 || true
    exit 1
  fi
  sleep 0.1
done

if [[ ! -S "$socket" ]]; then
  echo "Podman API socket did not become ready" >&2
  cat "$XDG_RUNTIME_DIR/podman/service.log" >&2 || true
  exit 1
fi

export LPIC_DAILY_RUN_PODMAN_INTEGRATION=1
go test -tags=integration ./internal/runner/podman   -run '^TestHostFilesystemSentinelSurvivesDestructiveLab$'   -count=1
