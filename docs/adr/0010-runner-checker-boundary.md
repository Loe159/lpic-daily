# ADR 0010 — Runner/checker boundary

Status: **Accepted — 2026-09-26**

## Decision

Keep lab execution and grading behind project-owned structured interfaces.

The domain runner accepts:
- an allowlisted lab definition;
- structured argv execution requests;
- explicit lifecycle operations;
- structured filesystem/process probes.

Checkers depend only on a read-oriented `Probe` interface, not on Podman/libvirt directly.

## Security consequences

- no generic host runner exists;
- no API accepts an arbitrary host mount;
- network mode is limited to `none` or `isolated`;
- capability profiles are project-defined allowlists;
- resource limits are mandatory;
- checkers read bounded data;
- grading observes final state rather than command history.

A caller may intentionally run `/bin/sh -c ...` **inside the sandbox** using structured argv when lesson setup/check logic genuinely requires shell semantics. The runner itself never concatenates user/content strings into a host shell command.

## Phase-1 capability profiles

Initial profiles are intentionally narrow:
- `baseline`: no extra capabilities;
- `identity-files`: `CHOWN`, `FOWNER`, `FSETID` (nécessaire pour poser SGID sur un répertoire appartenant à un autre groupe dans le user namespace rootless);
- `process-lab`: `KILL`.

The concrete Podman adapter must drop capabilities by default and map only these approved additions. Adding a capability profile requires security review.

## Adapter rule

Podman and libvirt implementations live behind `runner.Runner`. Neither the scheduler nor mastery/content packages import those vendor APIs.

This also gives ADR 0006 a concrete reversal path: if the Go Podman/libvirt libraries become unsuitable, adapters can be replaced without rewriting the learning domain.
