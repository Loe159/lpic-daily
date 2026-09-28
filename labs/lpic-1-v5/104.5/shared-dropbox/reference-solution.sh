#!/usr/bin/env bash
set -euo pipefail

chown root:project /srv/shared
chmod 3770 /srv/shared

for user in alice bob; do
    runuser -u "$user" -- /usr/bin/bash -c 'printf "umask 0007\\n" > "$1"' _ "/home/$user/.bash_profile"
done
