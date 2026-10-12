# ADR 0012 — Authored lab runtime boundary

Status: **Accepted — 2026-09-26**

## Decision

Built-in labs are shipped as authored data/scripts and loaded through a strict runtime layer.

The runtime:
- strictly decodes `lab.json` and hint JSON;
- resolves setup/reference paths only inside the lab directory;
- loads the setup script and hints;
- deliberately does **not** load the reference-solution script into normal runtime memory;
- compiles authored state checks to project-owned checker types;
- preserves each check-to-concept mapping through grading so a mixed lab attempt records pass/partial/fail evidence per concept instead of assigning one lab-wide result;
- requires every state check to map to one or more declared concept IDs and rejects labs with declared concepts that have no check evidence;
- maps the authored environment to `runner.Definition`;
- executes setup only through `runner.Exec`, inside the already-created sandbox.

## Lifecycle

```text
load authored lab
      |
      v
runner.Prepare
      |
runner.Start
      |
sandboxed setup via structured argv
      |
learner activity
      |
state checks
      |
runner.Destroy
```

Any prepare/start/setup failure triggers best-effort destruction with a bounded cleanup context.

## Reference solutions

A lab may carry `reference_solution_ref` for authoring, CI and future content verification. The normal loader validates that the referenced file exists but does not read its content. Graduated hints remain the learner-facing disclosure mechanism.

## Check types and fail-closed behavior

Schema evolution can get ahead of runtime support. If an authored check type is known to the schema but not implemented by the runtime, compilation fails closed.

The current runtime implements `file-exists`, `file-mode`, `file-owner`, `file-content-regex`, `process-running`, `process-absent`, `command-exit` and `block-device-state`. `command-exit` executes only through the selected sandbox/guest runner with structured argv; it is not a host-shell escape hatch.

## Security boundary

Authored shell scripts are untrusted content. They are never passed to a host shell. The setup body is passed as an argument to `/usr/bin/bash -eu -c` through the sandbox runner.


## Full-machine setup semantics

Phase-2 VM labs may use either `setup.execution_scope = "none"` or
`setup.execution_scope = "sandbox"`.

With `none`, the trusted image, firmware profile and disposable disks are the
complete initial state. With `sandbox`, the local setup script is executed
inside the disposable guest through `Runner.Exec`.

The libvirt runner now implements structured guest execution through QEMU Guest
Agent and the image-owned `/usr/libexec/qemu-ga/fsfreeze-hook.d/lpic-daily-exec`
helper. The runtime passes argv/environment as structured guest-agent fields,
rejects TTY/stdin/working-directory requests on this transport, bounds agent
responses and output, and never invokes a host shell. This satisfies the
transport boundary that previously required all libvirt labs to use `none`.

Podman labs continue to require `execution_scope = "sandbox"` plus a local
setup script.
