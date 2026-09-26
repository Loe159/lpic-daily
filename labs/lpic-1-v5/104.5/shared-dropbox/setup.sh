#!/usr/bin/env bash
set -euo pipefail

getent group project >/dev/null || groupadd --gid 2000 project
id alice >/dev/null 2>&1 || useradd --create-home --uid 1101 --groups project alice
id bob >/dev/null 2>&1 || useradd --create-home --uid 1102 --groups project bob

install -d -o root -g root -m 0755 /srv/shared
rm -f /srv/shared/*
