#!/usr/bin/env bash
set -euo pipefail
cd /workspace
cat < numbers.txt > stdin-copy.txt
./emit.sh > stdout.txt 2> stderr.txt
printf 'first\n' > append.txt
printf 'second\n' >> append.txt
grep -E '^[0-9]+$' numbers.txt | wc -l | tr -d ' ' > pipeline-count.txt
printf 'mirror\n' | tee tee-copy.txt > tee-captured.txt
printf 'alpha\nbeta\n' | xargs -n1 printf 'item:%s\n' > xargs.txt
./emit.sh > order-a.txt 2>&1
./emit.sh 2>&1 > order-b.txt
