#!/usr/bin/env bash
set -euo pipefail
cd /workspace
ln releases/v2/app source.hard
ln -s releases/v2/app source.sym
ln -s missing-target broken.sym
cp releases/v2/app app.copy
ln -s releases/v2 current
