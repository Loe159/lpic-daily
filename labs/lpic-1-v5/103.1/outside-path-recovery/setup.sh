#!/usr/bin/env bash
set -euo pipefail
install -d -m 0755 /run/lpic /opt/lpic/tools
rm -f /run/lpic/reconcile-ok
cat > /opt/lpic/tools/reconcile <<'EOF'
#!/usr/bin/env bash
set -euo pipefail
printf 'reconciled\n' > /run/lpic/reconcile-ok
EOF
chmod 0755 /opt/lpic/tools/reconcile
cat > /root/.bash_profile <<'EOF'
export PATH="/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin"
EOF
