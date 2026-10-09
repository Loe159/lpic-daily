#!/usr/bin/env bash
set -euo pipefail
mkdir -p /workspace/bin /workspace/vendor /workspace/data
printf 'report=green\n' > /workspace/data/report.txt
cat > /workspace/bin/run-report <<'EOF'
#!/usr/bin/env bash
set -euo pipefail
LD_PRELOAD=/workspace/vendor/libreport.so /usr/bin/cat /workspace/data/report.txt
EOF
chmod +x /workspace/bin/run-report
ln -s missing-libreport.so /workspace/vendor/libreport.so
