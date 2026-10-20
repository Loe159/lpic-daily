#!/usr/bin/env bash
set -euo pipefail
mkdir -p /etc/acme /var/log/acme /srv/acme /opt/acme/bin
mv /workspace/incoming/acme.conf /etc/acme/acme.conf
mv /workspace/incoming/service.log /var/log/acme/service.log
mv /workspace/incoming/index.html /srv/acme/index.html
mv /workspace/incoming/acmectl /opt/acme/bin/acmectl
