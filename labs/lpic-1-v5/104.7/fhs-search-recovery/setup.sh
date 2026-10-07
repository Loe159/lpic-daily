#!/usr/bin/env bash
set -euo pipefail
rm -rf /workspace/*
mkdir -p /workspace/staging /workspace/fhs /workspace/searchroot /workspace/bin
printf 'mode=production\n' > /workspace/staging/app.conf
printf 'started\n' > /workspace/staging/app.log
printf '#!/usr/bin/env bash\necho app\n' > /workspace/staging/app.bin
chmod 0755 /workspace/staging/app.bin
printf 'payload\n' > /workspace/staging/payload.dat
printf 'old\n' > /workspace/searchroot/needle-old.txt
cat > /workspace/bin/tool <<'EOF'
#!/usr/bin/env bash
echo tool
EOF
chmod 0755 /workspace/bin/tool
chmod -R a+rwX /workspace
