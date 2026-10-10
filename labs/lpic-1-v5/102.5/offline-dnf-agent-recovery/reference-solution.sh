#!/usr/bin/env bash
set -euo pipefail
sed -i 's|file:///srv/lpic-rpm-repo-old|file:///srv/lpic-rpm-repo|' /etc/yum.repos.d/lpic-offline.repo
dnf --disablerepo='*' --enablerepo='lpic-offline' --refresh -y install lpic-dnf-agent
