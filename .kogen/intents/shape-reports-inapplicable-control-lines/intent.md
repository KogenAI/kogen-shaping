# Show the failed hunk of an inapplicable controls

Slug: shape-reports-inapplicable-control-lines
Model: not applicable — bounded formatting of one refusal line has no meaningful state
Acceptance:
- .kogen/intents/shape-reports-inapplicable-control-lines/acceptance.feature
- features/steps/shape-reports-inapplicable-control-lines__steps.js
- features/steps/shape-reports-inapplicable-control-lines__driver/probe_test.go

When a patch in a controls overlay (the correct or a wrong implementation) does not apply to the candidate tree, the refusal line from shape/controls.go keeps its existing text as its first part, byte for byte, and then names the first hunk of the patch that failed (its header line). When the target file exists in the candidate tree, the line also shows up to 8 lines of that file starting at the hunk's old start line, each prefixed with its line number; lines past the end of the file are not shown or padded and no other file is read. When the target file does not exist in the candidate tree, the line says that the file does not exist in the candidate tree and shows no numbered lines. A patch that applies produces no refusal line, and which controls pass or fail is unchanged. For a wrong implementation the existing wording about the correct implementation being applied stays, and the excerpt shows the file as it is with the correct implementation applied. Out of scope: the shaper prompt, shape rounds and time budget, the request reviser, other controls refusals and documentation. Edit only shape/controls.go and shape/controls_test.go.

Acceptance fixtures: scenario 1 patches a 20-line file with a hunk whose context line 12 differs from the real file (notes.txt, hunk @@ -10,5 +10,5 @@); scenario 4 uses a 5-line file (short.txt, hunk @@ -3,3 +3,3 @@) whose hunk starts near its end, so only the 5-line file's existing lines are shown.

## Decisions
- The refusal stays one line: the existing text, then the failed hunk header, then the numbered excerpt or the missing-file statement, with the exact separators and number format left to the implementation. (evidence: shape/controls.go) (source: request)
- The failed hunk is the one whose old start line git apply reports as failed, else the first hunk; a patch with no hunk header adds nothing. (evidence: shape/controls.go) (assumption: git apply's 'patch failed: file:line' report names the hunk's old start line) (source: request)
- The excerpt is read from the overlay tree when the patch fails, so a wrong implementation sees the tree with the correct implementation applied. (decision: D3, round 1) (evidence: shape/controls.go; shape/controls_test.go) (source: request)
- The refusal stays one line: the existing text, then the failed hunk header, then the numbered excerpt or the missing-file statement; separators and number format are left to the implementation. (decision: D1, round 1) (constraint: existing text stays first byte for byte) (evidence: shape/controls.go) (source: request)
- The failed hunk is the one whose old start line git apply reports as failed, else the first hunk; a patch with no hunk header adds nothing. (decision: D2, round 1) (evidence: shape/controls.go) (assumption: git apply's 'patch failed: file:line' report was observed in a local scratch run to name the hunk's old start line; not retained as Kogen evidence.) (source: request)

## Source comparison
- Addressed: excerpt for a correct-implementation overlay (acceptance scenario 1), wrong-implementation overlay (scenario 2), missing file (scenario 3), hunk near the end of a short file (scenario 4), clean patch with no line (scenario 5).
- Addressed: existing refusal text kept first, byte for byte; the pass/fail judgment of controls is unchanged.
- Omitted: the shaper prompt, shape rounds, time budget, reviser and documentation, as the request excludes them.

## Tests
- A correct-implementation patch whose context differs reports the failed hunk header and eight numbered lines from its old start: acceptance scenario 1 and a focused test in shape/controls_test.go (integration)
- A wrong-implementation overlay keeps its wording and adds the same excerpt: acceptance scenario 2 (integration)
- A patch for a missing file says the file does not exist in the candidate tree: acceptance scenario 3 (integration)
- A hunk near the end of a short file shows only existing lines: acceptance scenario 4 (integration)
- A clean patch gives no refusal line: acceptance scenario 5 (integration)
- Existing patch overlay tests still pass: go test ./shape (existing)

## Feasibility
- evidence: {"id":"excerpt-observer","kind":"observer","inputs":[".kogen/intents/shape-reports-inapplicable-control-lines/controls.json"],"observers":["features/steps/shape-reports-inapplicable-control-lines__steps.js","features/steps/shape-reports-inapplicable-control-lines__driver/probe_test.go"],"checks":[{"name":"good","command":"node features/steps/shape-reports-inapplicable-control-lines__steps.js good","exit":0,"output":"observer accepted\n"},{"name":"bad","command":"node features/steps/shape-reports-inapplicable-control-lines__steps.js bad","exit":1,"output":"observer rejected\n"}]}
