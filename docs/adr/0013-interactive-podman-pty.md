# ADR 0013 — Interactive PTY over Podman exec hijack

Status: **Accepted — 2026-09-26**

## Decision

Interactive learner shells use a real pseudo-TTY allocated by Podman's Docker-compatible exec API.

The project does not invoke the `podman` CLI and does not allocate a host-side shell for the learner.

## Protocol

1. create an exec with stdin/stdout/stderr attached, `Tty=true`, `Privileged=false`;
2. open a second connection directly to the already-verified local rootless Podman Unix socket;
3. POST `/v1.40/exec/{id}/start` with HTTP upgrade headers;
4. require HTTP `101 Switching Protocols` and `Upgrade: tcp`;
5. forward the resulting raw PTY byte stream;
6. send terminal-size changes through `POST /v1.40/exec/{id}/resize?h=...&w=...`;
7. inspect the exec after the stream closes to obtain its exit code.

TTY output is deliberately **not** passed through Docker's stdout/stderr demultiplexer: a PTY is a single raw stream and stdout/stderr are merged by terminal semantics.

## Fail-closed rules

- non-TTY stdin remains unsupported until there is a concrete use case;
- invalid/zero terminal sizes are rejected;
- an upgrade other than `tcp` is rejected;
- failure of the initial resize aborts the interactive exec path;
- context cancellation closes the hijacked Unix connection;
- no error path executes anything on the host.

## Terminal rendering boundary

Raw PTY mode necessarily permits terminal control sequences because interactive programs require them. It is only appropriate for an explicit terminal passthrough surface. Future non-interactive TUI panes must sanitize untrusted guest output rather than rendering arbitrary control sequences.
