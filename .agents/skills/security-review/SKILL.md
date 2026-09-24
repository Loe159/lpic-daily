---
name: security-review
description: Review lab execution and privilege boundaries
---

# security-review


1. Read `docs/SECURITY.md`, `docs/THREAT_MODEL.md`, and ADR 0003.
2. Trace data/control flow from curriculum file to executed command/API call.
3. Assume curriculum/lab definitions can be malicious or malformed.
4. Check filesystem mounts, namespaces, Linux capabilities, devices, network access, secrets, host sockets and privilege escalation paths.
5. Verify no error path falls back to host execution.
6. Check teardown after crashes/timeouts and isolation between concurrent labs.
7. Prefer allowlisted structured operations over shell interpolation.
8. Record residual risks and required tests.

