# ADR 0014 — Structured command-exit lab checks

Status: **Accepted — 2026-09-26**

## Decision

Implement the existing `command-exit` content contract as a state/behavior checker executed through the assigned sandbox runner.

A command-exit check contains structured `argv` and an expected exit code. The checker calls `runner.Exec` directly; it never constructs or invokes a host shell.

## Rationale

Some Linux outcomes are best validated by asking the sandbox itself:

- whether a fresh login shell resolves a command through `PATH`;
- whether environment initialization produces a required value;
- whether a configuration is behaviorally valid rather than merely matching file text.

These cases should not force content authors to reverse-engineer every acceptable file representation.

## Security

- execution occurs only inside the already-created lab sandbox;
- argv remains structured at the runner boundary;
- no stdin or TTY is attached;
- exit code is the only evidence consumed by this checker;
- stdout/stderr are not trusted or rendered;
- authored commands remain subject to the lab's resource/time limits.

Using `/usr/bin/bash -lc` as an authored argv is allowed when shell semantics are the thing being tested. This is a **guest** shell inside the sandbox, never a host-shell fallback.

## Grouped exit-bitmask checks

When several criteria inspect the same sandbox state, consecutive `command-exit` entries may share `batch_id`, `argv` and `timeout_seconds`, with a unique power-of-two `failure_bit` per criterion. The trusted checker executes that argv **once** and maps each set exit-code bit to a failed criterion. Exit code 0 passes the entire group; timeouts or unexpected exit codes fail every criterion without granting mastery. Invalid groups fail closed.

This avoids repeatedly starting guest shells or attaching an interactive TTY while retaining separate learner-facing results and concept evidence. Keep the individual descriptions and source-level checks explicit in the authored lab. The batch remains subject to a short guest-side timeout and a host-side deadline.
