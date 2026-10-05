# ADR 0009 — Explainable Phase-1 session scheduler baseline

Status: **Accepted — 2026-09-26**

## Decision

Implement the Phase-1 scheduler as a pure deterministic domain function.

Inputs include:
- curriculum bundle;
- concept mastery projections;
- objective prerequisite readiness;
- active objective scope;
- current time;
- explicit policy values.

Output is an ordered session whose every item contains a machine-readable reason and a learner-facing explanation.

SQLite, TUI and runner code do not participate in selection logic.

## Prerequisite readiness

"Ready enough to unlock a dependant" is distinct from "objective complete".

The Phase-1 default readiness baseline is:
- concept stage threshold: `recall`;
- at least 70% of an objective's concepts meet that threshold.

For 103.1 (12 concepts), this currently means 9 concepts at recall or stronger.

This is deliberately a **tunable Phase-1 policy**, not a scientific claim or permanent requirement. Fine-grained concept prerequisites should replace coarse objective fractions where authored content demonstrates a real dependency.

## Review baseline

The initial review intervals are explicit policy values:
- exposed: 1 day;
- recall: 3 days;
- guided: 3 days;
- independent: 7 days;
- transfer: 21 days.

These numbers exist to make the vertical slice operational and testable. The evidence log is algorithm-independent, so a later scheduler can replace them without data migration.

## Session composition

Default Phase-1 session:
- up to 4 overdue reviews, oldest due first;
- up to 1 new concept whose hard prerequisites are ready;
- reviews precede new material.

Short/deep profiles can later supply different limits without modifying scheduling rules.

## Why not FSRS directly

LPIC Daily evidence includes labs, hints, transfer challenges and practical independence, not only flashcard response grades. Adopting a flashcard scheduler wholesale before mapping those evidence types would create false precision.

A future ADR may integrate FSRS or another model for retrieval timing while keeping practical mastery as a separate signal.

## Invariants

- objective readiness never marks remaining concepts as completed;
- new material respects hard prerequisites;
- recommended prerequisites affect future ranking, not eligibility;
- every selected item is explainable;
- scheduling never mutates evidence.
