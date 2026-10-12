#!/usr/bin/env bash
set -euo pipefail
printf 'export EDITOR=vi\n' > /root/.bashrc
cat > /tmp/lpic-ex <<'EOF'
/^port=/s/.*/port=8443/
/^mode=/s/.*/mode=production/
/^obsolete=true$/d
/^audit=enabled$/a
audit=required
.
wq
EOF
vi -es /workspace/app.conf < /tmp/lpic-ex
{
  command -v vi >/dev/null && echo 'vi=present' || echo 'vi=absent'
  command -v vim >/dev/null && echo 'vim=present' || echo 'vim=absent'
  command -v nano >/dev/null && echo 'nano=present' || echo 'nano=absent'
  command -v emacs >/dev/null && echo 'emacs=present' || echo 'emacs=absent'
} > /workspace/editor-inventory
