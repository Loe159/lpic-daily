#!/usr/bin/env bash
set -euo pipefail

getent group project >/dev/null
id alice >/dev/null 2>&1
id bob >/dev/null 2>&1

install -d -o root -g root -m 0755 /srv/shared
rm -f /srv/shared/*

for user in alice bob; do
    printf 'umask 0022\n' > "/home/$user/.bash_profile"
    chmod 0644 "/home/$user/.bash_profile"
done
