#!/usr/bin/env bash
set -euo pipefail
rm -rf /workspace/*
mkdir -p /workspace/releases/v2
printf 'version-2\n' > /workspace/releases/v2/app
chmod -R a+rwX /workspace
