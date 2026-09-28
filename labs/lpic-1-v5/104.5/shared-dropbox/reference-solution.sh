#!/usr/bin/env bash
set -euo pipefail

chown root:project /srv/shared
chmod 3770 /srv/shared

for user in alice bob; do
    printf 'umask 0007\n' > "/home/$user/.bash_profile"
    chown "$user:$user" "/home/$user/.bash_profile"
done
