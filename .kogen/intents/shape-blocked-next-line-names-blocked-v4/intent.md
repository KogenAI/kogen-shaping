# Name only blocked files in the Next advice

Slug: shape-blocked-next-line-names-blocked-v4
Model: not applicable — fixed report copy introduces no meaningful state
Acceptance:
- .kogen/intents/shape-blocked-next-line-names-blocked-v4/acceptance.feature
- features/steps/shape-blocked-next-line-names-blocked-v4__steps.js
- features/steps/shape-blocked-next-line-names-blocked-v4__driver/run.js
- features/steps/shape-blocked-next-line-names-blocked-v4__driver/report_test.go.txt

Make the blocked report's third line use exactly the Next sentence in acceptance.feature, referring to the files already listed rather than directing every reader to the project definition. Remove the obsolete sentence, without aliases or file-type-specific advice. Implementation scope is only shape/shape.go (the Next line in the blocked-report printer), shape/shape_test.go and shape/contract_test.go (the two output expectations), and features/steps/k-shape-blocked-report-repair__steps.js (the Next-line constant). Blocked-path validation, the first two output lines, exit codes, prompts, docs and every other implementation file are out of scope.

The inspected HEAD already contains the requested copy in the printer and in the repair step constant. Preserve correct implementation bytes rather than manufacturing an edit; this Intent supplies fresh independent verification of the complete requested outcome. (evidence: shape/shape.go; features/steps/k-shape-blocked-report-repair__steps.js)

## Decisions
- There is one such line for every blocked report; it does not change with the kind of file. Use the exact requested sentence, with no project-only or manifest-specific advice. (decision: D1, round 1) (evidence: shape/shape.go) (source: request)
- Preserve the three-line report, exit code 1 and absence of an Intent or approval. Restrict implementation edits to the four requested locations; where the inspected implementation already satisfies the request, retain it unchanged and verify it through independent v4-prefixed acceptance. (decision: D2, round 1) (evidence: shape/shape.go; features/steps/k-shape-blocked-report-repair__steps.js) (assumption: the current tree already holding the requested copy is accepted as the delivered state, so the acceptance may pass before any Build edit) (source: request)

## Source comparison
- Addressed: Request Expected and Acceptance 1–3 map to three full-output report scenarios in acceptance.feature: the protected settings glob, the project definition and the two dependency manifests. Each compares all three ordered lines exactly, exit 1, no Intent and no approval; the expected third line contains no project-definition pathname.
- Addressed: Request Acceptance 4 maps to the single-path and two-path repair journeys, run through the existing repair file's standalone program, plus a check of its constant against the requested sentence. The two existing Go tests that assert the line run in a compiled shape test binary.
- Addressed: supplied history (ev-5c8753d8, pattern blocked-next-line-names-wrong-file, six attempts across six rows) is retained in the feature description as supplied, not as new observation. controls.json supplies Go-output mutations: old sentence, project-only advice, manifest-specific appended sentence and repair-output disagreement.
- Superseded: the old advice directing all blocked reports to .kogen/project.yaml is replaced by the generic sentence; no compatibility form or protected-edit override.
- Omitted: nothing requested is omitted; out-of-scope validation, first two lines, exit codes, prompts and docs stay unchanged.

## Tests
- Fixed advice independent of file kind: compare complete reports from the real Shape entry point for the settings glob, project definition and dependency manifests (regression)
- Blocked report has no publication side effects: check exit 1 and absence of Intent and approval in each report fixture (integration)
- Repair feature uses the same exact copy: run its standalone single-path and two-path journeys and check the constant against the requested sentence (scenario)
- Existing Go expectations retain the requested output: run TestShapeBlockedReportRecognizesProtectedPathsMatchedByAGlob and TestShapeBlockedReportIsPrintedAndWritesNothing in the compiled shape test binary (existing)
- none: this is a fixed output-copy correction, not an operation over a collection of data records; multiple paths only exercise the supplied report format

## Feasibility
- evidence: {"id":"report-observer","kind":"observer","inputs":["features/steps/shape-blocked-next-line-names-blocked-v4__driver/report_test.go.txt","features/steps/k-shape-blocked-report-repair__steps.js"],"observers":["features/steps/shape-blocked-next-line-names-blocked-v4__driver/run.js"],"checks":[{"name":"good","command":"node features/steps/shape-blocked-next-line-names-blocked-v4__driver/run.js good","exit":0,"output":"accepted\n"},{"name":"bad","command":"node features/steps/shape-blocked-next-line-names-blocked-v4__driver/run.js bad","exit":1,"output":"rejected\n"}]}
