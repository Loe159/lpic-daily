#!/usr/bin/env bash
set -euo pipefail
mkdir -p /workspace/incoming /workspace/catalog/usr/share/doc
printf 'listen=8080\n' > /workspace/incoming/acme.conf
printf 'started\n' > /workspace/incoming/service.log
printf '<h1>Acme</h1>\n' > /workspace/incoming/index.html
cat > /workspace/incoming/acmectl <<'EOF'
#!/usr/bin/env bash
echo acme-ok
EOF
chmod 0755 /workspace/incoming/acmectl
printf 'Acme operations guide\n' > /workspace/catalog/usr/share/doc/acme-guide.txt
rm -f /workspace/acme.db /workspace/find-result.txt /workspace/command-report.txt
