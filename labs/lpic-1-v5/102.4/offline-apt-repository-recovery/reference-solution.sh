#!/usr/bin/env bash
set -euo pipefail
sed -i 's|file:/opt/lpic-broken-repo|file:/opt/lpic-repo|' /etc/apt/sources.list.d/lpic-local.list
apt-get update
DEBIAN_FRONTEND=noninteractive apt-get install -y lpic-alert-agent
