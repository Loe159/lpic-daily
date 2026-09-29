#!/usr/bin/env bash
set -euo pipefail

if [[ "${EUID}" -eq 0 ]]; then
  echo "Run Phase-2 acceptance as the regular LPIC Daily user, not root." >&2
  exit 1
fi
if [[ "${LPIC_DAILY_RUN_KVM_INTEGRATION:-}" != "1" ]]; then
  echo "Set LPIC_DAILY_RUN_KVM_INTEGRATION=1 to acknowledge the real KVM/libvirt acceptance run." >&2
  exit 1
fi

python3 scripts/validate_foundation.py

dirty="$(find . -name '*.go' -type f -print0 | xargs -0 gofmt -l)"
if [[ -n "${dirty}" ]]; then
  echo "gofmt required:" >&2
  echo "${dirty}" >&2
  exit 1
fi

go mod tidy
if ! git diff --quiet -- go.mod go.sum; then
  echo "go mod tidy changed go.mod/go.sum" >&2
  git diff -- go.mod go.sum >&2
  exit 1
fi

go test ./...
go vet ./...
go run ./cmd/lpic validate
go run ./cmd/lpic doctor

go test -tags=integration ./internal/runner/libvirt -count=1 -v
