#!/usr/bin/env bash
set -euo pipefail
cd /workspace
grep '^alpha [0-9][0-9]*$' corpus.txt > bre.txt
grep -E '^(alpha|beta) [0-9]+$' corpus.txt > ere.txt
grep -E '^[[:upper:]]+' corpus.txt > classes.txt
grep -E '^delta [0-9]{2}$' corpus.txt > quantified.txt
grep -E '^(beta [0-9]+|gamma [[:alpha:]]+)$' corpus.txt > groups.txt
grep -in alpha corpus.txt > search.txt
sed 's/^beta /BETA /' corpus.txt > sed.txt
(cd glob && printf '%s\n' report?.log | sort) > glob.txt
find glob -maxdepth 1 -type f -printf '%f\n' | grep -E '^report[0-9]+\.log$' | sort > regex.txt
