#!/usr/bin/env bash
set -euo pipefail
mkdir -p /etc/acme /var/log/acme /srv/acme /opt/acme/bin
mv /workspace/incoming/acme.conf /etc/acme/acme.conf
mv /workspace/incoming/service.log /var/log/acme/service.log
mv /workspace/incoming/index.html /srv/acme/index.html
mv /workspace/incoming/acmectl /opt/acme/bin/acmectl
updatedb -l 0 -U /workspace/catalog -o /workspace/acme.db
find /workspace/catalog -type f -name 'acme-guide.txt' > /workspace/find-result.txt
PATH="/opt/acme/bin:$PATH"
{
  printf 'type=%s\n' "$(type -P acmectl)"
  printf 'which=%s\n' "$(which acmectl)"
  printf 'whereis=%s\n' "$(whereis bash)"
} > /workspace/command-report.txt
