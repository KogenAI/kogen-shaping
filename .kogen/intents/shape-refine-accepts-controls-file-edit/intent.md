# Let Refine edit the Intent's controls file

Slug: shape-refine-accepts-controls-file-edit
Model: not applicable — the change is a stateless ownership rule for which submitted paths Refine accepts
Acceptance:
- .kogen/intents/shape-refine-accepts-controls-file-edit/acceptance.feature
- features/steps/shape-refine-accepts-controls-file-edit__steps.js

During `kogen intent refine`, the `controls.json` of the Intent being refined (`.kogen/intents/<slug>/controls.json`) becomes an editable draft file, like the Intent's step files. A submission that replaces its whole content is accepted when the complete candidate then passes the same checks a Shape submission passes, including the run of every wrong-implementation control against the stored controls. A refused controls edit keeps the retained draft's earlier controls content. A submission that edits any other path outside the owned files is refused with the same text as today, and the file stays unchanged.

The refusal text keeps its existing sentence "not an exclusively owned editable draft file" and additionally names the refused path and lists the owned files, so the model learns what it may submit. Only `shape/refine.go` (the owned-file list and the refusal text), `shape/refine_test.go` and `shape/prompts/refine.md` (to list the controls file among the editable files) change. Shape's own rounds, the Build baseline guard, the publication scan, docs and README stay as they are.

## Decisions
- The controls file of the refined Intent is added to the owned editable files of Refine, beside intent.md, acceptance.feature and model.qnt. (decision: D1, round 1) (evidence: shape/refine.go) (source: request)
- A controls edit is accepted only when the complete candidate passes the same checks as a Shape submission, including the wrong-implementation controls run against the stored controls; a refused edit leaves the earlier controls content in place. (decision: D2, round 1) (evidence: shape/controls.go; shape/shape.go) (source: request)
- Any other path outside the owned files is refused with the existing refusal text. (decision: D3, round 1) (evidence: shape/refine.go) (source: request)
- The refusal text keeps the existing sentence and also contains the refused path and the owned file paths of the Intent, including its intent.md, acceptance.feature and controls file. (decision: D4, round 1) (evidence: shape/refine.go) (source: request)

## Source comparison
- Addressed: the controls file is editable during Refine (request, Expected). Addressed: the refusal text names the refused path and the owned files (the "outside" scenario asserts the path and the owned paths). Addressed: controls run on a controls edit (acceptance 2). Retained: other paths such as go.sum are refused with the existing sentence (acceptance 3). Omitted by scope: Shape rounds, Build baseline guard, publication scan, docs, README.

## Tests
- A replacement controls file without a home path is accepted and the draft carries it: scripted Refine session on a fixture draft whose controls hold a home path (scenario)
- A replacement controls file whose wrong-implementation control no longer fails the wrong implementation is refused and the earlier controls stay: scripted Refine session (scenario)
- An edit of go.sum is refused with the existing text, the refused path and the owned files, and the file is unchanged: scripted Refine session (scenario)
- The existing protected and shared step refusals keep passing: shape/refine_test.go (existing)

## Feasibility
- none
