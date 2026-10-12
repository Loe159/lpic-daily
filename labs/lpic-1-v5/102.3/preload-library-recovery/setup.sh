#!/usr/bin/env bash
set -euo pipefail
mkdir -p /workspace/bin /workspace/vendor /workspace/data
printf 'report=green\n' > /workspace/data/report.txt
cat > /workspace/bin/run-report <<'EOF'
#!/usr/bin/env bash
set -euo pipefail
preload=/workspace/vendor/libreport.so
# glibc may ignore a missing LD_PRELOAD with exit code 0. Demand actual ELF loading.
if ! LD_PRELOAD="$preload" /usr/bin/ldd /usr/bin/cat 2>/dev/null | /usr/bin/grep -Fq "$preload ("; then
  printf 'required preload library was not loaded: %s\n' "$preload" >&2
  exit 70
fi
LD_PRELOAD="$preload" /usr/bin/cat /workspace/data/report.txt
EOF
chmod +x /workspace/bin/run-report
ln -s missing-libreport.so /workspace/vendor/libreport.so
