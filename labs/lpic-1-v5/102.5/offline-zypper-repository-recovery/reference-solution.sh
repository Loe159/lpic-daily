#!/usr/bin/env bash
set -euo pipefail
sed -i 's|file:///srv/lpic-zypp-repo-missing|file:///srv/lpic-zypp-repo|' /etc/zypp/repos.d/lpic-maint.repo
zypper --non-interactive --no-gpg-checks refresh lpic-maint
zypper --non-interactive --no-gpg-checks --no-refresh install --from lpic-maint lpic-archive-agent
test "$(/usr/local/bin/lpic-archive-agent)" = 'zypp-agent=ready'
