#!/usr/bin/env bash
set -euo pipefail
mkdir -p /root/lpic-zypper
zypper --non-interactive rr lpic-local >/dev/null 2>&1 || true
zypper --non-interactive ar -G file:///opt/lpic/rpmrepo lpic-local
zypper --non-interactive refresh lpic-local
zypper --non-interactive install --from lpic-local -y lpic-zypper-demo
zypper lr -u > /root/lpic-zypper/repos.txt
zypper se -si lpic-zypper-demo > /root/lpic-zypper/search.txt
zypper info lpic-zypper-demo > /root/lpic-zypper/info.txt
zypper --non-interactive remove -y lpic-zypper-demo
zypper --non-interactive install --from lpic-local -y lpic-zypper-demo
