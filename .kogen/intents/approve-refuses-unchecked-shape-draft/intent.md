# Require passed Shape checks before approval

Slug: approve-refuses-unchecked-shape-draft
Model: not applicable — a persisted pass record is checked against a package hash and a request digest, with no interacting state machine
Acceptance:
- .kogen/intents/approve-refuses-unchecked-shape-draft/acceptance.feature
- features/steps/approve-refuses-unchecked-shape-draft__steps.js

A Shape draft is approvable only when Shape recorded that its checks (the `contract_review` and readiness) passed for exactly this package and this request. An interrupted `kogen intent shape` (SIGTERM or SIGINT after the shaper's package was written to the worktree, before the contract review and readiness finished) leaves the Intent draft on disk with no not-ready latch. Kogen then treats that draft as shaped: a second `kogen intent shape` for the slug answers `intent/exists` ("is already shaped", exit 5), and `kogen intent approve` prints the review card with an approval hash and approves it with that hash, although no Shape gate (contract review, readiness) ever passed for that draft. The interrupted Shape itself returns the interrupt (exit 130).

On the ready path (end of `shapeRun` in shape/shape.go) Shape records one pass record for the slug, stored under `.kogen/local/` with the other per-slug local records, binding: the package hash (`project.Hash` of `project.Baseline`, the hash `approve` already computes and prints), a digest of the request text captured in the source receipt, and that the `contract_review` and readiness passed. Refine's successful path (`RecordPass` in shape/refine.go) records the same for the package it produces, with the request digest of the source receipt, so a refined Intent stays approvable.

A Shape draft is an Intent whose slug has a shaping record directory `.kogen/local/shaping/<slug>/`. A Shape draft whose current package hash and captured request do not match a pass record is not ready:
- Shape does not answer `intent/exists` for it; it treats it like a latched draft (the documented recovery: retry Shape on the retained draft, or delete `.kogen/intents/<slug>/` of the slug and shape again), so a second Shape sends shaper requests on the retained draft.
- `kogen intent approve`, without and with a hash, refuses it with `*sys.Error` class `intent`, reason `not_ready`, exit 1, a detail saying that no passed Shape checks are recorded for this package, and the `draftRecovery` lines. It prints no card and no `Approve with:` line and writes no approval marker.

An Intent with no shaping record directory at all (written by hand, as the approve and contract test fixtures do) keeps today's behaviour exactly. A latched draft is refused exactly as today with its existing text. The binding is the package hash and the request, not HEAD: an approved-on-an-older-base Intent whose package is unchanged keeps today's behaviour. No new configuration, flag or environment variable. The existing "already shaped" subtest in shape/shape_test.go writes intent.md with no shaping record and keeps answering `intent/exists`; update it only as far as the rule requires.

## Decisions
- The pass record is bound to the package hash plus a digest of the captured request text, written by Shape's ready path and Refine's success path; presence alone never qualifies a draft. (decision: D1, round 1) (constraint: an edited package or a different request makes the draft not ready) (evidence: shape/shape.go; shape/refine.go; shape/approve.go) (assumption: The exact storage location and file name of the pass record are left to the Build; a probe in a scratch copy only showed the unfixed tree approves an interrupted draft.) (source: request)
- Approve and Shape treat a draft with a shaping record directory but no matching pass record as not ready, reusing the not_ready error shape and draftRecovery lines; the latch stays unchanged. (constraint: no new flags, settings or latch changes) (evidence: shape/draft_refusal.go; shape/approve.go) (source: request)
- Acceptance injects a focused Go journey test into the shape package at acceptance time using the existing scripted-provider fixture, with a shpStep handler that calls `f.S.Interrupt("SIGINT")`. (evidence: shape/helper_test.go; shape/shape_test.go) (assumption: A scratch run on the unfixed tree reproduced the bug (card, hash approval and intent/exists after an interrupt); the red state with the feature implemented was not observed.) (source: request)
- Approve and Shape treat a draft with a shaping record directory but no matching pass record as not ready, reusing the not_ready error shape and draftRecovery lines; the latch stays unchanged. (decision: D2, round 1) (constraint: no new flags, settings or latch changes) (evidence: shape/draft_refusal.go; shape/approve.go) (assumption: none beyond the request) (source: request)
- Acceptance injects a focused Go journey test into the shape package at acceptance time using the existing scripted-provider fixture. (decision: D3, round 1) (evidence: shape/helper_test.go; shape/shape_test.go) (assumption: A scratch run on the unfixed tree reproduced the bug (card, hash approval and intent/exists after an interrupt); the red state with the feature implemented was not observed.) (source: request)

## Source comparison
- Addressed: interrupted Shape leaves a not-ready draft, refused by approve without and with hash and retried by a second Shape (scenario 1).
- Addressed: a draft edited after its checks passed is not approvable, and restoring the file restores the card (scenario 2).
- Addressed: checks recorded for a different request do not count even with a byte-identical package (scenario 3).
- Addressed: unchanged behaviour for ready drafts, `intent/exists` on an unchanged passed Intent with no model request, and approval by the printed hash (scenario 4). Refine, latched drafts and Intents without a shaping record are preserved by the existing tests (shape refine tests, shape/shape_test.go latch tests, contract/approve_test.go).
- Omitted: the kogen-loops dispatcher workaround in bin/run-item.sh is outside this repository.

## Tests
- Interrupted Shape leaves an unapprovable draft that Shape retries: injected TestUncheckedDraftJourney scenario 1 (scenario)
- A draft edited after its checks passed is not approvable: injected journey scenario 2 (scenario)
- A pass for a different request does not count for a byte-identical package: injected journey scenario 3 (scenario)
- Ready drafts keep card, approval and `intent/exists`: injected journey scenario 4 (regression)
- Refine stays approvable, latched drafts and Intents without a shaping record unchanged: the existing refine, latch and approve tests (existing)

## Feasibility
- none
