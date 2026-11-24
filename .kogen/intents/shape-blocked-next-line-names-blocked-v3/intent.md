# Name only blocked files in the Next advice

Slug: shape-blocked-next-line-names-blocked-v3
Model: not applicable — fixed report copy introduces no meaningful state
Acceptance:
- .kogen/intents/shape-blocked-next-line-names-blocked-v3/acceptance.feature
- features/steps/shape-blocked-next-line-names-blocked-v3__steps.js
- features/steps/shape-blocked-next-line-names-blocked-v3__driver/run.js
- features/steps/shape-blocked-next-line-names-blocked-v3__driver/report_test.go.txt

Make the blocked report's third line use exactly the Next sentence in acceptance.feature, referring to the files already listed rather than directing every reader to the project definition. Remove the obsolete sentence, without aliases or file-type-specific advice. Implementation scope is only shape/shape.go (the Next line in the blocked-report printer), shape/shape_test.go and shape/contract_test.go (the two output expectations), and features/steps/k-shape-blocked-report-repair__steps.js (the Next-line constant). Blocked-path validation, the first two output lines, exit codes, prompts, docs and every other implementation file are out of scope. New v3 Intent and acceptance artifacts specify and verify this bounded outcome, not an expansion of implementation scope.

The inspected HEAD already contains the requested copy in all four locations. Preserve correct implementation bytes rather than manufacturing an edit; this Intent supplies fresh independent verification of the complete requested outcome. The two affected existing Go tests already pass. (evidence: shape/shape.go; shape/shape_test.go; shape/contract_test.go; features/steps/k-shape-blocked-report-repair__steps.js)

## Decisions
- There is one such line for every blocked report; it does not change with the kind of file. Use the exact requested sentence, with no project-only or manifest-specific advice. (decision: D1, round 1) (evidence: shape/shape.go) (source: request)
- Preserve the three-line report, exit code 1 and absence of an Intent or approval. Restrict implementation edits to the four requested locations; where the inspected implementation already satisfies the request, retain it unchanged and verify it through independent v3-prefixed acceptance. (decision: D2, round 1) (evidence: shape/shape.go; shape/shape_test.go; shape/contract_test.go; features/steps/k-shape-blocked-report-repair__steps.js) (source: request)

## Source comparison
- Addressed: Request Expected and Acceptance 1–3 map to the three full-output report scenarios in acceptance.feature, including the protected settings glob, automatic project-definition protection and the two dependency manifests. The scenarios compare all three ordered lines exactly, exit 1, no Intent and no approval; the expected third line contains no project-definition pathname.
- Addressed: Request Acceptance 4 maps to the single-path and two-path repair journey examples. Independent v3 bindings run the existing repair file as the subject under test through its standalone program boundary, without reusing its Cucumber bindings, and check both its real journey result and its constant. The existing-Go-expectations scenario runs the two existing tests named by the inspected source. Local fixture conventions come from shape/helper_test.go; the runner and check commands remain those in .kogen/project.yaml and README.md.
- Addressed: The supplied historical evidence ev-713f0ef2, pattern blocked-next-line-names-wrong-file, four attempts across four rows since 19 Nov, named rows and reported paths are retained in the feature description as supplied history, not new observations. The executable report assertions reject the old fixed sentence, project-only old advice and appended manifest advice; the repair checks reject a stale constant. controls.json supplies actual Go-output mutations for these report cases and repair disagreement. Its correct overlay retains the already-correct contract-test bytes. Its repair mutation changes output rather than the step constant because controls.go forbids controls overlays in the steps directory; the constant itself remains explicitly checked by acceptance and the qualified observer's bad controls.
- Superseded: The old advice directing all blocked reports to .kogen/project.yaml is replaced by the exact requested generic sentence in precisely the four scoped locations. No compatibility form or protected-edit override is introduced. The current HEAD has already performed that replacement, so no unrelated production work is authorized.
- Omitted: No requested behavior is omitted. Out-of-scope validation, initial lines, exit codes, prompts, docs, configuration and other implementation files remain unchanged. No prior Intent approval or readiness is reused.

## Tests
- Fixed advice independent of file kind: compare complete reports from the real Shape entry point for the settings glob, project definition and dependency manifests (regression)
- Blocked report has no publication side effects: check exit 1 and absence of Intent and approval in each report fixture (integration)
- Repair feature uses the same exact copy: run its standalone single-path and two-path journeys and check the constant against the requested sentence (scenario)
- Existing Go expectations retain the requested output: run TestShapeBlockedReportRecognizesProtectedPathsMatchedByAGlob and TestShapeBlockedReportIsPrintedAndWritesNothing in the compiled shape test binary (existing)
- none: this is a fixed output-copy correction, not an operation over a collection of data records; multiple paths only exercise the supplied report format

## Feasibility
- Both affected existing Go tests passed with real verbose PASS markers in the current environment; the requested copy is already implemented. (evidence: ran go test ./shape -run '^(TestShapeBlockedReportRecognizesProtectedPathsMatchedByAGlob|TestShapeBlockedReportIsPrintedAndWritesNothing)$' -count=1 -v)
- evidence: {"id":"report-observer","kind":"observer","inputs":["features/steps/shape-blocked-next-line-names-blocked-v3__driver/report_test.go.txt","features/steps/k-shape-blocked-report-repair__steps.js"],"observers":["features/steps/shape-blocked-next-line-names-blocked-v3__driver/run.js"],"checks":[{"name":"good","command":"node features/steps/shape-blocked-next-line-names-blocked-v3__driver/run.js good","exit":0,"output":"accepted\n"},{"name":"bad","command":"node features/steps/shape-blocked-next-line-names-blocked-v3__driver/run.js bad","exit":1,"output":"rejected\n"}]}
