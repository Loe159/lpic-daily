#!/usr/bin/env bash
set -euo pipefail

chown root:project /srv/shared
chmod 3770 /srv/shared

rm -f /srv/shared/team-note
(
    umask 0007
    : > /srv/shared/team-note
)
