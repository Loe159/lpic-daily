#!/usr/bin/env bash
set -euo pipefail
podman rm -f lpic-worker
podman run --pull=never -d --name lpic-worker --network=none --pid=private --read-only localhost/lpic-guest-probe:1 /probe
test "$(podman exec lpic-worker /probe --kernel)" = "$(uname -r)"
