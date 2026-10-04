# Contributing to LPIC Daily

LPIC Daily accepts focused changes that preserve its learning, traceability and sandbox-security contracts.

## Before starting

Read `README.md`, `docs/PRODUCT.md` and `docs/REQUIREMENTS.md`. For agent-assisted work, also read `AGENTS.md` and the relevant nested instructions/skills.

Create work from `main` and keep one pull request focused on one coherent change.

## Requirements

- preserve objective and stable concept-ID traceability;
- never add a host-execution fallback for learner commands;
- keep the normal application unprivileged;
- use state-based grading instead of matching one exact command sequence;
- update tests, coverage and documentation when behavior changes;
- keep educational content original rather than copying/adapting LPI Learning Materials.

## Validate

Run the checks relevant to the change. The normal baseline is:

```bash
python3 scripts/validate_foundation.py
go mod tidy
git diff --exit-code -- go.mod go.sum
test -z "$(find . -name '*.go' -type f -print0 | xargs -0 gofmt -l)"
go test ./...
go vet ./...
go run ./cmd/lpic validate
```

Changes to Podman/libvirt code should also run their integration/acceptance coverage on a compatible host.

## Pull requests

Explain:
- what changed and why;
- affected objective/concept IDs when applicable;
- security/isolation impact;
- validations actually executed;
- documentation or migration implications.

Agent-specific contribution guidance is documented in `docs/CONTRIBUTING_WITH_AGENTS.md`.
