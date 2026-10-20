#!/usr/bin/env bash
set -euo pipefail
mkdir -p /workspace/incoming
printf 'listen=8080\n' > /workspace/incoming/acme.conf
printf 'started\n' > /workspace/incoming/service.log
printf '<h1>Acme</h1>\n' > /workspace/incoming/index.html
cat > /workspace/incoming/acmectl <<'EOF'
#!/usr/bin/env bash
echo acme-ok
EOF
chmod 0755 /workspace/incoming/acmectl
