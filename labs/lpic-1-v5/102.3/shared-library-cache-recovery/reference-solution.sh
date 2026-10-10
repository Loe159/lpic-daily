#!/usr/bin/env bash
set -euo pipefail
printf '%s\n' /opt/lpic/vendor/lib > /etc/ld.so.conf.d/lpic-metrics.conf
ldconfig
