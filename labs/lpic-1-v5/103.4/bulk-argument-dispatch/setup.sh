#!/usr/bin/env bash
set -euo pipefail
mkdir -p /workspace/input/data /workspace/output /workspace/bin
printf '12345\n' > /workspace/input/data/alpha.txt
# The length of one input file changes, so a hardcoded result will not pass.
seed="$(od -An -N2 -tu2 /dev/urandom | tr -d '[:space:]')"
printf '%0*d\n' "$((12 + seed % 9))" 0 > /workspace/input/data/beta.txt
printf 'abc\n' > /workspace/input/data/gamma.txt
printf '%s\n' /workspace/input/data/gamma.txt /workspace/input/data/alpha.txt /workspace/input/data/beta.txt > /workspace/input/targets.txt
cat > /workspace/bin/describe-file <<'EOF'
#!/usr/bin/env bash
set -eu
for path in "$@"; do
  printf '%s:%s\n' "$(basename "$path")" "$(wc -c < "$path")"
done
EOF
chmod 0755 /workspace/bin/describe-file
