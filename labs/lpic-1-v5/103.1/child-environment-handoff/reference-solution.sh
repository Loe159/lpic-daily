#!/usr/bin/env bash
set -euo pipefail

source /root/.bashrc
export REPORT_ENV
unset REPORT_DEBUG
start-report-worker
