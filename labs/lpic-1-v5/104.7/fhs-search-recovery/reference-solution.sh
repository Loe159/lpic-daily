#!/usr/bin/env bash
set -euo pipefail
cd /workspace
mkdir -p fhs/etc/lpic fhs/var/log/lpic fhs/usr/local/bin fhs/srv/lpic
cp staging/app.conf fhs/etc/lpic/app.conf
cp staging/app.log fhs/var/log/lpic/app.log
cp staging/app.bin fhs/usr/local/bin/app
cp staging/payload.dat fhs/srv/lpic/payload.dat
updatedb -U /workspace/searchroot -o /workspace/locate.db
printf 'new\n' > /workspace/searchroot/needle-new.txt
find /workspace/searchroot -maxdepth 1 -type f -name 'needle-*.txt' -printf '%f\n' | sort > find-result.txt
locate -d /workspace/locate.db needle- | sed 's#.*/##' | sort > locate-before.txt
updatedb -U /workspace/searchroot -o /workspace/locate.db
locate -d /workspace/locate.db needle- | sed 's#.*/##' | sort > locate-after.txt
whereis bash > whereis.txt
PATH=/workspace/bin:$PATH
type -t tool > type.txt
which tool > which.txt
