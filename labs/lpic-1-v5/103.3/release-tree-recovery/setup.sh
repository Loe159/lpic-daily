#!/usr/bin/env bash
set -euo pipefail
mkdir -p /workspace/source/app/bin /workspace/source/app/conf /workspace/source/app/logs /workspace/source/app/cache
printf 'deploy-v2\n' > /workspace/source/app/bin/deploy.sh
chmod 0755 /workspace/source/app/bin/deploy.sh
printf 'mode=prod\n' > /workspace/source/app/conf/app.conf
printf 'dsn=primary\n' > /workspace/source/app/conf/db.conf
printf '012345678901234567890123456789\n' > /workspace/source/app/logs/current.log
printf 'ok\n' > /workspace/source/app/logs/small.log
printf 'stale\n' > /workspace/source/app/cache/old.tmp
printf 'fresh\n' > /workspace/source/app/cache/new.tmp
touch -d '10 days ago' /workspace/source/app/cache/old.tmp
touch -d '1 day ago' /workspace/source/app/cache/new.tmp
