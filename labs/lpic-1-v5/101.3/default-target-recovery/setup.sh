#!/usr/bin/env bash
set -euo pipefail
test -d /run/systemd/system
systemctl set-default rescue.target
