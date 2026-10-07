#!/usr/bin/env bash
set -euo pipefail
cd /workspace
wc -l < input.txt | tr -d ' ' > line-count.txt
od -An -tx1 input.txt > octets.txt
sed -n '1,2p' input.txt | cut -d: -f1 > selected.txt
cut -d: -f1 input.txt | sort | uniq > unique.txt
paste left.txt right.txt > pasted.txt
split -l 1 -d -a 1 left.txt chunk-
tr '[:lower:]' '[:upper:]' < input.txt | sed 's/ALPHA/A/g' > transformed.txt
sha256sum input.txt > sha256.txt
zcat input.txt.gz > from-gzip.txt
bzcat input.txt.bz2 > from-bzip2.txt
xzcat input.txt.xz > from-xz.txt
