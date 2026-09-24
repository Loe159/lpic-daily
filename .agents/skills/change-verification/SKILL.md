---
name: change-verification
description: Verify a repository change before declaring it complete
---

# change-verification


1. Identify the files and contracts affected by the change.
2. Read the nearest `AGENTS.md` and governing ADRs.
3. Run the smallest relevant validators/tests first, then broader tests if available.
4. Check documentation and machine-readable coverage for drift.
5. Review the diff for accidental scope expansion, secrets, unsafe fallbacks, and stale comments.
6. Report commands executed, results, and any unverified assumptions. Never claim a check you did not run.

