# ADR 0004 — Code and content licensing

Status: **Accepted — 2026-09-24**

## Decision
LPIC Daily permits commercial reuse.

Use:
- **Apache License 2.0** for application code, scripts and code-like schemas/examples unless a file states otherwise;
- **Creative Commons Attribution 4.0 International (CC BY 4.0)** for original educational prose, curriculum explanations and project documentation unless a file states otherwise.

Third-party material retains its original license and must be tracked explicitly.

## Constraint
Official LPI Learning Materials are not a source to copy or adapt into this curriculum because their published licensing includes NonCommercial and NoDerivatives restrictions. Use official LPIC objectives as factual scope, then write original teaching material from independent technical research and primary Linux documentation.

## Rationale
- Commercial reuse is explicitly acceptable to the product owner.
- Apache-2.0 is permissive and includes an explicit patent grant suitable for an open-source software project.
- CC BY 4.0 allows broad reuse, including commercial reuse and adaptations, while requiring attribution.
- Separating software and educational-content licenses makes reuse expectations clearer.

## Consequences
- Repository contributions must be compatible with these license boundaries.
- Imported snippets/assets require provenance and license review.
- A future contributor guide must explain whether a file is code or content when the classification is not obvious.
