#!/usr/bin/env bash
set -euo pipefail
id=$(cat /var/lib/lpic-grub-recovery/entry-id)
grub2-set-default "$id"
# Faire :reboot dans le terminal parent avant de vérifier le boot réel.
