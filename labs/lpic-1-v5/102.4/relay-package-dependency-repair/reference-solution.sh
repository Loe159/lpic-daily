#!/usr/bin/env bash
set -euo pipefail
dpkg -i /opt/lpic-debs/lpic-relay-runtime_1.0_all.deb
dpkg --configure lpic-relay-agent
