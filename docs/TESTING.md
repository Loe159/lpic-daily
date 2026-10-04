# Testing strategy

## Validation layers

### Foundation and authored content

`python3 scripts/validate_foundation.py` validates the complete LPIC objective inventory, prerequisite graph, concept inventory, schemas, authored references and Phase-1 coverage invariants.

The Go curriculum/content loaders independently validate embedded authored JSON against the Draft 2020-12 schemas before semantic validation. Invalid or unsupported content fails closed.

### Phase 3 curriculum coverage

`python3 scripts/audit_phase3_coverage.py --check` validates Phase-3 artifact mappings without requiring unfinished Exam-101 content to be complete. `--require-complete` is the theory/daily-question/mastery-advancement exit gate. The audit reads the canonical `phase3-exam101.json` manifest and, like the Go runtime and embedded filesystem, discovers lesson/question JSON recursively below their content directories.

The study planner only schedules Phase-3 objectives whose mapped concepts all have a focused `introduce` lesson, at least one daily question, and a path beyond recognition: either a daily recall-capable question or a practical lab. `initial-assessment` questions do not count toward this gate. Tests verify 103.4 becomes reachable after 103.1 readiness, unfinished objectives remain unavailable, and concepts with multiple questions recommend the least-attempted question.

### Go unit and domain tests

`go test ./...` covers the curriculum/content loaders, scheduler and mastery projection, SQLite progress store, TUI behavior, runner contracts, state checkers, lab orchestration and CLI behavior.

`go vet ./...` and `gofmt` are enforced in CI. `go mod tidy` must leave `go.mod` and `go.sum` unchanged.

### Podman integration and security

Phase-1 Podman integration tests compile on every CI run and execute against a real rootless Podman service on both Ubuntu and Fedora.

These tests cover the authored Phase-1 lab lifecycle and security invariants, including fail-closed backend handling, resource/capability restrictions and destructive host-sentinel isolation checks.

### TUI and terminal safety

Non-interactive TUI rendering is tested for deterministic behavior and sanitization of untrusted terminal control characters.

Raw PTY passthrough is reserved for explicit interactive terminal surfaces where terminal control sequences are required.

### Lab conformance

Authored labs are validated against their schemas and runtime contracts. Loading a built-in lab also validates the derived runner definition and compiles checker configuration (including Go regex syntax), so invalid authored execution settings fail before the lab is advertised as runnable. Phase-1 coverage generation requires all 22 selected concepts to remain traceable, to have exactly one focused `introduce` lesson, at least one daily non-assessment question, and two distinct machine-checked practical contexts.

State-based grading must accept equivalent valid end states rather than depending on an exact learner command transcript.

## CI tiers

The current GitHub Actions workflow runs:

- foundation/schema/coverage validation on Ubuntu;
- Go formatting, tests, vet and embedded-content validation on Ubuntu;
- Go tests and embedded-content validation on Fedora 44;
- real rootless Podman Phase-1 conformance/security tests on Ubuntu;
- real rootless Podman Phase-1 conformance/security tests on Fedora 44.

Fast PR CI must not require KVM.

## Phase 2 verification

The libvirt/QEMU/KVM backend has unit and non-KVM CI coverage and the real-KVM integration layer is implemented, but Phase 2 is not accepted until that suite has actually executed successfully on a compatible host.

Remaining acceptance-level execution includes:

- real KVM/libvirt lifecycle and host-sentinel testing;
- base-image immutability verification;
- isolated-network no-forwarding plus host-bridge blocking verification;
- boot/reboot coverage for the 102.2 reference lab;
- end-to-end 104.1 and 102.2 execution using the released/reproducible guest-image pipeline.

Hypervisor scenarios belong on a KVM-capable runner or dedicated integration environment. Never weaken VM isolation or verification merely to make generic hosted CI pass.


### Phase 2 real-host acceptance command

On a provisioned KVM/libvirt host with the trusted Fedora image installed, run:

```bash
LPIC_DAILY_RUN_KVM_INTEGRATION=1 scripts/run_phase2_acceptance.sh
```

This entry point runs the normal foundation/Go/vet/format gates, then the opt-in libvirt integration suite. The real-host suite uses the same VM state path as the application, verifies private peer communication, failed public egress, failed access to the host uplink and failed access to the host-side bridge gateway, preserves a host sentinel and the immutable base image, exercises crash reaping, and executes the 104.1 and reboot-dependent 102.2 reference solutions end to end.
