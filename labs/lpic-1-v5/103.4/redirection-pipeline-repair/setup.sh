#!/usr/bin/env bash
set -euo pipefail
rm -rf /workspace/*
cat > /workspace/emit.sh <<'EOF'
#!/usr/bin/env bash
printf 'OUT\n'
printf 'ERR\n' >&2
EOF
chmod 0755 /workspace/emit.sh
printf '10\nnoise\n20\n30x\n40\n' > /workspace/numbers.txt
chmod -R a+rwX /workspace
