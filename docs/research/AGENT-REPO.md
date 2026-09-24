# Agent-first repository research synthesis

## Design rule
The repository itself should be the long-lived context, not chat history. Give agents a short entry contract, then let them discover stable documentation, ADRs, machine-readable manifests and reusable skills just in time.

## Applied practices
- Root `AGENTS.md` contains only project-wide invariants and navigation.
- Nested `AGENTS.md` files scope rules to curriculum, labs and application code.
- `.agents/skills/` encodes repeated workflows such as authoring, security review and verification.
- Vendor files (`CLAUDE.md`, `GEMINI.md`, Copilot instructions) are adapters, not duplicate handbooks.
- ADRs distinguish durable decisions from temporary execution plans.
- Canonical machine-readable coverage prevents prose drift.
- Validation scripts convert important assumptions into checks agents can run.
- Research sources are dated so a future agent knows what needs freshness verification.

## Anti-patterns intentionally avoided
- one enormous AGENTS.md containing all architecture/domain details;
- conflicting duplicated instructions across agent vendors;
- relying on TODO comments/chat messages for architectural decisions;
- prose-only coverage claims that CI cannot audit;
- asking agents to infer security boundaries from implementation accidents.
