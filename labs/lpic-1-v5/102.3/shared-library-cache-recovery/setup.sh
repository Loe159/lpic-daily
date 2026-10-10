#!/usr/bin/env bash
set -euo pipefail
command -v gcc >/dev/null
mkdir -p /opt/lpic/vendor/lib
cat > /tmp/lpic-metrics.c <<'EOF'
int lpic_metric(void) { return 73; }
EOF
gcc -shared -fPIC -Wl,-soname,liblpicmetrics.so.1 -o /opt/lpic/vendor/lib/liblpicmetrics.so.1.0 /tmp/lpic-metrics.c
ln -s liblpicmetrics.so.1.0 /opt/lpic/vendor/lib/liblpicmetrics.so.1
cat > /tmp/lpic-report.c <<'EOF'
#include <stdio.h>
extern int lpic_metric(void);
int main(void) { if (lpic_metric() != 73) return 1; puts("metrics=ready"); return 0; }
EOF
gcc -o /usr/local/bin/lpic-report /tmp/lpic-report.c -L/opt/lpic/vendor/lib -l:liblpicmetrics.so.1.0
printf '%s\n' /opt/lpic/vendor/old-lib > /etc/ld.so.conf.d/lpic-metrics.conf
ldconfig
if env -i PATH=/usr/bin:/bin /usr/local/bin/lpic-report >/dev/null 2>&1; then
  echo 'Expected missing ELF dependency' >&2; exit 1
fi
