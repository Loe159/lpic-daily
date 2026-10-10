#!/usr/bin/env bash
set -euo pipefail
state=/var/lib/lpic-grub-legacy
source="$(cat "$state/source-entry")"
id="$(cat "$state/import-id")"
target="/boot/loader/entries/$id.conf"
cp "$source" "$target"
sed -i \
  -e 's/^title .*/title LPIC imported legacy recovery/' \
  -e '/^options /s/$/ lpic.legacy_import=1/' \
  "$target"
grub2-set-default "$id"
# :reboot dans le terminal parent est requis pour valider le démarrage réel.
