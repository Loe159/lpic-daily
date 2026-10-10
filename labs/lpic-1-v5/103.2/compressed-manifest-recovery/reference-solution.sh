#!/usr/bin/env bash
set -euo pipefail
sha256sum /workspace/input/day1.txt.gz /workspace/input/day2.txt.bz2 /workspace/input/day3.txt.xz > /workspace/input/manifest.sha256
{
  zcat /workspace/input/day1.txt.gz
  bzcat /workspace/input/day2.txt.bz2
  xzcat /workspace/input/day3.txt.xz
} > /workspace/output/audit.txt
{
  printf 'MD5 %s\n' "$(md5sum /workspace/output/audit.txt | awk '{print $1}')"
  printf 'SHA256 %s\n' "$(sha256sum /workspace/output/audit.txt | awk '{print $1}')"
  printf 'SHA512 %s\n' "$(sha512sum /workspace/output/audit.txt | awk '{print $1}')"
} > /workspace/output/checksums.txt
