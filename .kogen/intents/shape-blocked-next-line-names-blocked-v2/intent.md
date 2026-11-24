# Name only blocked files in the Next advice

Slug: shape-blocked-next-line-names-blocked-v2
Model: not applicable — this is a fixed output-copy correction with no new state
Acceptance:
- .kogen/intents/shape-blocked-next-line-names-blocked-v2/acceptance.feature
- features/steps/shape-blocked-next-line-names-blocked-v2__steps.js
- features/steps/shape-blocked-next-line-names-blocked-v2__driver/run.js
- features/steps/shape-blocked-next-line-names-blocked-v2__driver/report_test.go.txt

Replace the fixed third line of the blocked-report printer with the exact Next sentence in acceptance.feature. It must refer to the files already listed rather than sending all readers to the project definition. Remove the old sentence completely from the four in-scope implementation/expectation locations. Implementation scope is only shape/shape.go (the Next line), shape/shape_test.go and shape/contract_test.go (their old-line expectations), and features/steps/k-shape-blocked-report-repair__steps.js (the old-line constant). The new Intent and its acceptance artifacts specify and verify that change; they do not expand implementation scope. Blocked-path validation, the first two output lines, exit codes, prompts, docs and every other implementation file remain unchanged.

## Decisions
- There is one such line for every blocked report; it does not change with the kind of file. Use exactly the requested Next sentence, without project-specific or manifest-specific advice. (decision: D1, round 1) (evidence: shape/shape.go) (source: request)
- Preserve the existing three-line report, exit code 1 and absence of an Intent or approval; change only the four requested sentence/expectation locations. The existing Shape fixtures and the repair program provide the local execution boundaries. (decision: D2, round 1) (evidence: shape/shape.go; shape/shape_test.go; shape/contract_test.go; shape/helper_test.go; features/steps/k-shape-blocked-report-repair__steps.js) (source: request)

## Source comparison
- Addressed: Request Expected and Acceptance 1–3 map to the three exact-output report scenarios in acceptance.feature, including settings/check.cfg protected by a glob, the automatic .kogen/project.yaml baseline, and cli/go.mod plus cli/go.sum. Each compares the complete ordered three-line output, exit 1, no Intent and no approval. The exact new third line contains no .kogen/project.yaml.
- Addressed: Request Acceptance 4 maps to both repair journey examples, running the existing repair program as the subject under test through its standalone executable boundary, not importing its Cucumber bindings. Our independent slug-prefixed bindings check successful real journeys and the changed constant. The affected-existing-tests scenario verifies both requested Go expectation updates.
- Addressed: The report comparisons reject the supplied old-sentence, project-only and longer-manifest counterexamples; the repair constant comparison rejects the supplied stale-constant counterexample. Native implementation controls are deferred because their mechanical gate forbids overlays of features/steps/k-shape-blocked-report-repair__steps.js even though changing that existing step constant is the requested behavior; a correct overlay cannot make every scenario pass without that edit. This does not defer or weaken the requested implementation or its acceptance. Historical evidence ev-713f0ef2, the pattern name, four attempts/four rows since 19 Nov, named rows and reported paths are retained in the feature description, not presented as newly observed runs.
- Superseded: The old fixed advice to review and commit .kogen/project.yaml is replaced in precisely the four locations named in Request Scope. There is no alias, conditional advice or protected-edit override.
- Addressed: Request out-of-scope restrictions remain explicit in the outcome above. No requested behavior is omitted or deliberately broadened. README testing conventions and the configured Cucumber runner are retained unchanged. (evidence: README.md; .kogen/project.yaml)

## Tests
- Fixed advice for every blocked file kind: compare full report output for the settings glob, project definition and two dependency manifests through the real Shape entry point with a scripted shaper (regression)
- Unchanged blocked-report effects: assert exit 1, no Intent and no approval in those same fixtures (integration)
- Repair feature agrees with the changed copy: run the existing standalone repair journeys for single-path mixed and two-path protected reports and check their new constant (scenario)
- Existing Go expectations agree with the changed copy: run TestShapeBlockedReportRecognizesProtectedPathsMatchedByAGlob and TestShapeBlockedReportIsPrintedAndWritesNothing through the compiled shape test binary (existing)
- none: this is fixed report copy, not an operation processing a collection of records; the two-path case only covers the supplied output example

## Feasibility
- The existing passing test's real verbose output uses the PASS marker consumed by the harness. (probe: go test ./shape -run '^TestShapeBlockedReportIsPrintedAndWritesNothing$' -count=1 -v; exit 0)
- The independent overlay fixture reached the real Shape entry point on the unchanged tree and observed the old third line, exit 1 and no Intent or approval for the project path. (probe: local driver project observation; exit 0)
- evidence: {"id":"report-observer","kind":"observer","inputs":["features/steps/shape-blocked-next-line-names-blocked-v2__driver/report_test.go.txt","features/steps/k-shape-blocked-report-repair__steps.js"],"observers":["features/steps/shape-blocked-next-line-names-blocked-v2__driver/run.js"],"checks":[{"name":"good","command":"node features/steps/shape-blocked-next-line-names-blocked-v2__driver/run.js good","exit":0,"output":"accepted\n"},{"name":"bad","command":"node features/steps/shape-blocked-next-line-names-blocked-v2__driver/run.js bad","exit":1,"output":"rejected\n"}]}
