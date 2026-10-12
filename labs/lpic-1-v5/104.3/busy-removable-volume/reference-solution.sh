#!/usr/bin/env bash
set -euo pipefail
systemctl stop lpic-transfer-reader.service
umount /srv/transfer
