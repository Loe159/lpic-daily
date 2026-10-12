#!/usr/bin/env bash
set -euo pipefail
good=/srv/lpic-recovery/lpic-ledger-watch-1.0-1.noarch.rpm
gpgv --keyring /etc/lpic-recovery/trustedkeys.gpg "$good.asc" "$good"
rpm -K --nosignature "$good"
rpm -Uvh --replacepkgs "$good"
rpm -V lpic-ledger-watch
test "$(/usr/local/bin/lpic-ledger-watch)" = ledger=ready
