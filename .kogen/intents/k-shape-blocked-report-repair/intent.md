# Make refused blocked reports repairable

Slug: k-shape-blocked-report-repair
Model: not applicable — a stateless validation message: the refusal text is a function of the reported paths and the protected set computed for that submission
Acceptance:
- .kogen/intents/k-shape-blocked-report-repair/acceptance.feature
- features/steps/k-shape-blocked-report-repair__steps.js

The shaper can submit the exclusive `blocked` report (a truthful report that a required edit touches a protected path) for paths that are not protected at all. Today `validateBlocked` (shape/shape.go) refuses each such path with only `blocked path <p> is not protected by the current contract`. It does not say that an unprotected path is freely editable by the builder (so there is no conflict and no blocked report is needed). It does not list the protected set, although validateBlocked already computes it with `project.Baseline(sh.root, sh.slug, sh.cfg)`. It does not say the next move.

After this change, a `blocked` report that lists at least one non-empty text path outside that protected set is refused with repairable feedback. The refusal still goes through the existing `invalid submission:` tool output and the existing retry budget. These lines come after the existing envelope problem lines (`blocked.paths ...`, `blocked.reason ...`), in this order:

1. One line for each reported path not in the protected set, in report order. Protected paths in the same report get no line:
   `blocked path <p> is not protected by the current contract: the builder may edit it, so it is no reason to block`
2. One line showing the exact protected set the check used, formatted with the existing `InspectStrings` (the same list form the Protection contract context prints), in `project.Baseline` order:
   `protected paths under the current contract (configured protect patterns and automatic baseline): [".kogen/project.yaml", "config/deploy.cfg"]`
   With no configured protect patterns this list is the automatic baseline, for example `[".kogen/project.yaml"]`, never an empty list.
3. One next-move line:
   - when none of the reported text paths is protected: `none of the reported paths is protected: withdraw the blocked report and submit the Intent files`
   - when at least one reported path is protected: `keep only the protected paths in the blocked report, with its reason, or withdraw the report and submit the Intent files if no protected edit is required`

Example, with protect pattern `config/*.cfg` matching `config/deploy.cfg` and a report of `deploy/service.yaml`, `calc.js`, `README.md`. The complete tool output is:

```
invalid submission:
blocked path deploy/service.yaml is not protected by the current contract: the builder may edit it, so it is no reason to block
blocked path calc.js is not protected by the current contract: the builder may edit it, so it is no reason to block
blocked path README.md is not protected by the current contract: the builder may edit it, so it is no reason to block
protected paths under the current contract (configured protect patterns and automatic baseline): [".kogen/project.yaml", "config/deploy.cfg"]
none of the reported paths is protected: withdraw the blocked report and submit the Intent files
No proposals from this submission were retained; earlier proposals are unchanged.
```

Unchanged:
- A report whose problems involve no unprotected text path (empty or non-list paths, blank or non-text paths, a missing or non-text reason) keeps exactly its existing problem lines and gets no protected-set or next-move line. For example, empty paths with reason `7` yield exactly `blocked.paths must be a non-empty list` and `blocked.reason must be non-empty text`. A blank path `" "` yields exactly `blocked.paths must contain non-empty paths`.
- A report of only protected paths with a non-empty reason is accepted as today. Shape prints `shape/protected_conflict: required edits touch protected files: ...`, the reason and the existing `Next:` line, and exits 1. It writes no Intent, approval, run or project file, makes no commit, and authorizes no edit.
- After a refusal the shaper may submit files normally, and they go through the normal checks.
- The check stays model-free and uses the same protected set as today. Protection is not weakened, and `blocked` does not become an override.
- The Shape context prompt (shape/context.go) and the blocked-report output are not changed.
- Not added: requiring each reported protected path to be tied to an identified required edit (the request marks it optional).

The protected acceptance driver `features/steps/k-shape-blocked-report-repair__steps.js` embeds a Go test. It compiles that test into the shape test binary as `shape/k_shape_blocked_report_repair_acceptance_test.go` through `go test -c -vet=off -overlay` and never writes it into the checkout. The Build must not create a file with that name, or define identifiers starting with `kbr` or the test `TestKShapeBlockedReportRepairAcceptance` in package shape. It must keep the existing fixture APIs the driver uses usable: `newShpFx`, `shpSubmit`, `shpFiles`, `shpUI`, `shpOutputs`, `shpShaperRequests`, `shpStep`, and the `shpFx` methods `write`, `git`, `head`, `exists`, `read`, `setReport` and `shape`, plus `Prov.requests`.

## Decisions
- A refusal names each reported path outside the protected set as unprotected and editable by the builder, so it is no reason to block. It shows the protected set the check used and states the next move. Withdraw the report and submit the package when none of the listed paths is protected. (source: request)
- In a mixed report, only the unprotected paths are named. A resubmitted report of the protected path with a reason is accepted. (source: request)
- Protected-only reports with a reason behave as today: no files are written and no edit is authorized. (source: request)
- Problem lines for empty paths and a non-text reason are unchanged. (source: request)
- With no protect patterns, the feedback lists the automatic baseline, not an empty list. (source: request)
- The check stays model-free, does not weaken protection and does not make `blocked` an override. Tying reported paths to required edits is not added. (source: request)

## Source comparison
- Addressed: request Problem bullet 1 (shape/shape.go validateBlocked's one-line refusal, with the protected set already computed by `project.Baseline`) and the Need. The editable-path line, protected-set line and next-move line are specified above. They are exercised by acceptance scenarios "unprotected" (three unprotected paths, then an accepted files submission) and "automatic".
- Addressed: Synthetic acceptance item 1. The "unprotected" scenario uses protect pattern `config/*.cfg`, reports `deploy/service.yaml`, `calc.js` and `README.md`, then scripts a files submission that is published with exit 0.
- Addressed: item 2. The "mixed" scenario reports `calc.js`, `config/deploy.cfg` and `README.md`. The refusal names only the two unprotected paths. A resubmitted report of `config/deploy.cfg` with a reason is accepted (exit 1, exact protected_conflict stdout, nothing written).
- Addressed: item 3. The "protected" scenario is accepted at the first request, with exact stdout. It leaves no Intent, approval or run, no HEAD change, a clean git status and unchanged protected bytes.
- Addressed: item 4. The "malformed" scenario checks the exact existing output for empty paths with a non-text reason (7), and for a blank path.
- Addressed: item 5. The "automatic" scenario has no protect patterns, and the set line lists `[".kogen/project.yaml"]`.
- Omitted by request: the optional rule tying each reported path to a required edit. The request says it is not required here, and it would need judgment the model-free check cannot make.
- Not changed: shape/context.go's protection guidance (request Problem bullet 2). The Need asks only for refusal feedback. Changing the prompt would also change prompt goldens without a stated need.
- Not changed: the "protected but not required" direction (Problem bullet 3) stays accepted. The Need does not ask for a check, and the check stays model-free.

## Tests
- Each unprotected reported path gets the editable line, followed by the protected-set line and the none-protected next move: update the "blocked paths that are not protected" row of TestShapeRejectsInvalidSubmissions and the "unprotected" case of TestShapeBlockedSubmissionsNeedAnActualProtectedPathAndAReason to expect the complete new lines; acceptance scenario "unprotected" (integration)
- A mixed report names only its unprotected paths and gives the keep-only-protected next move, and the corrected protected-only report is accepted: add a case to the blocked validation tests in shape/contract_test.go; acceptance scenario "mixed" (integration)
- Without protect patterns the protected-set line lists the automatic baseline `[".kogen/project.yaml"]`: acceptance scenario "automatic" and the updated TestShapeRejectsInvalidSubmissions row, which uses the default fixture with no patterns (integration)
- Protected-only reports with a reason are accepted, print the existing protected_conflict output and write nothing: TestShapeBlockedReportIsPrintedAndWritesNothing, TestShapeBlockedPathsMayBeConfiguredGlobs and TestShapeBlockedReportRecognizesProtectedPathsMatchedByAGlob, plus acceptance scenario "protected" (existing)
- Empty paths, blank paths and non-text reasons keep exactly their existing problem lines with no added lines: existing TestShapeRejectsInvalidSubmissions rows plus acceptance scenario "malformed" (existing)
- Unrelated shaping behavior is unchanged: the full configured format/compile/vet/go-test checks; regenerate synthetic goldens only with `go test ./... -update` if any affected golden records this feedback (existing)

## Feasibility
- Repository inspection: the change is confined to shape/shape.go validateBlocked. submitOne already returns the validation problems as `invalid submission:` plus the problem lines plus the no-retention sentence. `project.Baseline` (kogen-coordinator project/baseline.go) returns the Intent files, Acceptance paths, `.kogen/project.yaml` and pattern matches in that order. With no patterns and no Intent directory it returns `[".kogen/project.yaml"]`. No protected path needs an edit, and no external protocol or library is involved, so no documentation lookup was needed.
- Scratch probe of the embedded journey on the unchanged tree, through real Cucumber (`testdata/tools/cucumber`) and the go -overlay test binary: 5 scenarios, 2 passed ("protected", "malformed") and 3 failed ("unprotected", "mixed", "automatic"). Each red scenario failed on exactly one assertion, the old one-line refusal text. Their later assertions (the files submitted after a refusal are published with exit 0; the corrected mixed report is accepted with exit 1, exact stdout and nothing written) already held. (probe: exit 1, about 117 s, mostly building the test binary)
- The driver compiles a generated Go test and judges each journey with several assertions. Its judge is qualified by controls that use the same submitted bytes. `good` judges one correct observation per mode and must accept all of them. `bad` judges 16 known-bad observations, each breaking one assertion (the old refusal text, an empty set, the wrong next move, a protected path named as unprotected, a refused follow-up, writes, commits, protected edits, extra malformed lines, wrong exits). It exits 1 only when every one of them is rejected. Observed in scratch: good printed `accepted` with exit 0; bad printed `rejected` with exit 1.
- evidence: {"id":"blocked-report-observer","kind":"observer","inputs":["features/steps/k-shape-blocked-report-repair__steps.js"],"observers":["features/steps/k-shape-blocked-report-repair__steps.js"],"checks":[{"name":"good","command":"\"$KOGEN_NODE\" features/steps/k-shape-blocked-report-repair__steps.js good","exit":0,"output":"accepted\n"},{"name":"bad","command":"\"$KOGEN_NODE\" features/steps/k-shape-blocked-report-repair__steps.js bad","exit":1,"output":"rejected\n"}]}
