# Phase 1 mastery projection

Status: **implementation baseline, intentionally revisable**

The product contract fixes the qualitative stages and invariants. Phase 1 now has a minimal deterministic projection so scheduler/storage code can be tested without inventing a percentage score.

## Canonical stages

`unseen -> exposed -> recall -> guided -> independent -> transfer`

The raw source of truth remains append-only evidence events. The stage is a projection.

## Mapping

- lesson exposure -> `exposed`;
- recognition-style answer -> at most `exposed`;
- free recall/fill-in -> `recall`;
- guided practical success -> `guided`;
- practical success with no or only a light hint -> `independent`;
- transfer task -> `transfer` only after earlier independent evidence in a different activity context.

A level 2+ hint caps an otherwise independent/transfer attempt at `guided`. Revealing the solution also caps it at `guided`.

Failures and partial attempts are retained for scheduling/explanation but do not erase a previously demonstrated stage.

## Experimental transfer gap

The Phase-1 default policy requires **24 hours** between the prior independent evidence and a transfer event.

This number is not a claim of scientific optimality and is not a frozen product requirement. It is an explicit, testable baseline. The projection API accepts a policy so later experimentation can change the gap without changing stored evidence.

## Why recognition is not recall

Multiple-choice success can often be driven by recognition. LPIC Daily deliberately keeps it weaker than free retrieval so a user cannot reach strong mastery by repeating recognition questions.

## Future work

The scheduler may later derive a decaying confidence/due date from failures, recency and evidence strength. That must remain separate from the monotonic historical stage: forgetting risk changes what should be reviewed, not what evidence happened in the past.
