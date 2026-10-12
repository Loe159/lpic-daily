#!/usr/bin/env bash
set -euo pipefail
command -v cloud-init >/dev/null
install -d -m 0755 /var/lib/cloud/seed/nocloud /etc/cloud/cloud.cfg.d
printf 'datasource_list: [ NoCloud ]\n' > /etc/cloud/cloud.cfg.d/99-lpic-nocloud.cfg
cat > /var/lib/cloud/seed/nocloud/meta-data <<'EOF'
instance-id: lpic-recovery-2026
local-hostname: lpic-recovery
EOF
cat > /var/lib/cloud/seed/nocloud/user-data <<'EOF'
#cloud-config
write_file:
  - path: /etc/lpic-cloud/agent.conf
    owner: root:root
    permissions: '0640'
    content: |
      agent=ready
EOF
rm -f /etc/lpic-cloud/agent.conf
cloud-init clean --logs
cloud-init init --local
cloud-init init
cloud-init modules --mode=config
cloud-init modules --mode=final
test ! -e /etc/lpic-cloud/agent.conf
