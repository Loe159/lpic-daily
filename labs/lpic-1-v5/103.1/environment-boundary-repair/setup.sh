#!/usr/bin/env bash
set -euo pipefail
install -d -m 0755 /run/lpic
cat > /run/lpic/repair-environment.sh <<'EOF'
# Ce fichier sera source dans un shell avec trois variables polluees.
# Repare les exports, supprime la variable obsolete et produis env-summary.
EOF
rm -f /run/lpic/env-summary /run/lpic/.environment-check-state
