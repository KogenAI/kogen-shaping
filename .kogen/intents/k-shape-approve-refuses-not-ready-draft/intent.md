# Refuse approval of not-ready Shape drafts

Slug: k-shape-approve-refuses-not-ready-draft
Model: .kogen/intents/k-shape-approve-refuses-not-ready-draft/model.qnt invariants=RefusedNeverApproved seed=41 samples=100 steps=20
Acceptance:
- .kogen/intents/k-shape-approve-refuses-not-ready-draft/acceptance.feature
- features/steps/k-shape-approve-refuses-not-ready-draft__steps.js
- features/steps/k-shape-approve-refuses-not-ready-draft__harness.js

A draft left by Shape ending intent/not_ready must not be approvable, even if its files are edited or readiness independently passes later. Persist the failed outcome and the last refusing check(s) alongside the retained draft. Both approval forms must return a typed error naming those checks and recovery. Only successful Shape clears the refusal. Allow the named Shape retry on the retained failed draft; alternatively the caller may delete the draft folder and shape again. Do not add a bypass. Ready drafts and hand-written never-shaped drafts retain current approval behavior.

Use persistent local status associated with the slug and draft lifecycle, not the current content hash: changing files cannot erase failure history. Consult it on every Approve path before showing a card or accepting any hash, including an earlier printed stale hash. Keep all draft artifacts on refusal. A successful retry clears the status only after all Shape gates succeed. Deleting the Intent folder and shaping anew starts a new draft lifecycle; unrelated lifecycle outcomes are not newly classified as not_ready by this request.

The intent/not_ready and intent/exists diagnostics list every file left by the draft, including its acceptance and step files. Explain both recovery choices and end both diagnostics with the same named next command: kogen intent shape <slug> <request-file>, substituting the actual slug. Approval refusal names that command as well. Update delivered tests that encode superseded diagnostics or refusal behavior in the same implementation; do not weaken tests of preserved behavior.

## Decisions
- Persist a slug-associated failed-Shape latch independent of draft content hashes; only successful Shape clears it for a retained draft. (source: request)
- Use the existing intent/not_ready typed refusal category for both approval forms, with check names, recovery command and no newly written approval marker. (evidence: shape/approve.go) (source: request)
- Preserve the no-hash review card's exit 5 with nil error and matching-hash approval's exit 0 and marker publication for ready or never-shaped packages. (evidence: shape/approve.go; shape/approve_test.go) (source: request)
- Update existing assertions of the old intent/exists wording and exact not_ready lines, rather than treating those old diagnostic expectations as protected product behavior. (evidence: shape/shape_test.go) (source: request)

## Source comparison
- Addressed: request Problem and Need, including shape/shape.go:752 and shape/approve.go approveLocked, by the persistent refusal and approval gate above and the sticky refusal scenario.
- Addressed: synthetic acceptance 1 and 5 by approval without a hash, with the current hash, with an earlier printed hash, and after a hand edit; each must refuse with no approval marker.
- Addressed: synthetic acceptance 2 by a successful second scripted Shape run followed by the normal review card and hash approval.
- Addressed: synthetic acceptance 3 by the never-shaped scenario, preserving the existing review card rather than inventing an error for it. (evidence: shape/approve_test.go)
- Addressed: synthetic acceptance 4 by the diagnostic scenario, requiring the same final recovery command and complete retained file lists.
- Addressed: delivery note by reading current source and tests before preserving behavior, and explicitly requiring implementation updates to changed-behavior tests. No protected-baseline exception or recovery of the previous Build is requested.
- Omitted: no requested behavior. Existing unrelated Shape, Refine, approval, source-custody and readiness rules remain in force.

## Tests
- Failed Shape blocks every approval route despite hand edits: scripted Shape and Approve regression journey (regression)
- Successful retry clears the refusal and restores normal card and approval: same-repository scripted Shape journey (integration)
- Never-shaped and ready packages preserve current approval semantics: observed card and marker publication (integration)
- Refusal and exists diagnostics identify all artifacts and the recovery command: inspect typed diagnostic output (scenario)
- Diagnostic behavior replaces old exact expectations: update TestShapeUndefinedStepGoesBackToTheShaperAndStaysNotReady and the already-shaped precondition test, plus other delivered assertions of changed output as needed (regression)

## Feasibility
- Quint model typechecked successfully. (probe: exit 0, 0.3 s) (evidence: ran `"$KOGEN_NODE" "$KOGEN_QUINT" typecheck model.qnt`)
