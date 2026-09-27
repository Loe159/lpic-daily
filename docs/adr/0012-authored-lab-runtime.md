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

## Unsupported check types

Schema evolution can get ahead of runtime support. If an authored check type is known to the schema but not implemented by the runtime, compilation fails closed. Phase 1 currently does this for `command-exit`.

## Security boundary

Authored shell scripts are untrusted content. They are never passed to a host shell. The setup body is passed as an argument to `/usr/bin/bash -eu -c` through the sandbox runner.


## Full-machine setup semantics

Phase-2 VM labs may use `setup.execution_scope = "none"`. In that mode the
trusted image, firmware profile and disposable disks are the complete initial
state and the runtime performs no guest command setup.

Podman labs continue to require `execution_scope = "sandbox"` plus a local
setup script. Until a separately authenticated structured guest transport is
implemented, libvirt labs are required to use `none`; this prevents the
generic lab lifecycle from accidentally treating a VM like a container and
calling `Runner.Exec`.
