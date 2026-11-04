# Unpin consumer protected-path expectations

Slug: k-shape-protect-list-unpinned
Model: not applicable — test expectation maintenance changes no workflow state
Acceptance:
- .kogen/intents/k-shape-protect-list-unpinned/acceptance.feature
- features/steps/k-shape-protect-list-unpinned__steps.js

Make kogen-shaping tests and goldens consume the coordinator's protected-baseline API or assert only the paths relevant to the test. Never pin its complete automatic baseline, including through version-specific conditional lists. Keep shaping production behavior and the current dependency pin unchanged. Preserve meaningful assertions about configured protection, blocked-path validation, approval cards, context, and wire compatibility; normalize only the coordinator-owned variable protected-list portion of synthetic goldens, not unrelated prompt or tool bytes. Do not routinely regenerate the force or policy compatibility goldens to hide this difference.

The reported regression is K-SHAPE-PROTECT-LIST-UNPINNED: coordinator 4826fb1 (Add opt-in model lineup files, K-MODELS-FILE) intentionally adds the models file to its baseline. The report measured shaping main 8c5ea31: d8feb1f passed, while 4826fb1 and 482d565 failed, including shape_test.go:567's calc.js blocked diagnostic. Current checkout pins c4a49a4fe84f and already contains some models-file conditionals; those are not an acceptable general solution.

Full delivery requires all three full-suite commands below, not just the bounded regression acceptance. Run each before completion (each invokes go test -count=1 ./... with isolated coordinator selection):

```
node features/steps/k-shape-protect-list-unpinned__steps.js --suite pinned
node features/steps/k-shape-protect-list-unpinned__steps.js --suite 482d565
node features/steps/k-shape-protect-list-unpinned__steps.js --suite 482d565 .kogen/future-baseline.yaml
```

## Decisions
- Consumer-first: repair tests and any affected golden, not shaping behavior or coordinator protection. (source: request)
- Support both today's pinned coordinator and main 482d565 or later, plus future automatic baseline additions. (source: request)

## Source comparison
- addressed: request Need and Acceptance are covered by bounded regression scenarios and the mandatory full-suite matrix above: pinned, 482d5656e610487a96b382770cbf276491697abc, and that version with an additional automatic baseline path.
- addressed: named regressions TestApproveCardOutput, TestContextBlocksCarryTheContractsAndEarlierRounds, TestShapeBlockedSubmissionsNeedAnActualProtectedPathAndAReason, TestShapeForceWireCompatibility, and TestShapeMixedBlockedReportNamesOnlyUnprotectedPathsAndTheCorrectedReportIsAccepted are exercised by acceptance; also cover TestShapeSubmissionEnvelopeProblems and inspect contract_test.go's exact-list expectations.
- addressed: historical revisions and calc.js diagnostic above are reproducing evidence, not authority to freeze a replacement complete list. All consumer expectations must follow the API or scoped assertions.
- superseded: the initial proposal's full-suite-per-scenario route exceeds readiness's 120-second limit; bounded regression scenarios do not replace the mandatory full-suite delivery matrix.
- omitted: no requested behavior is deliberately omitted; no production behavior, policy, protection settings, or dependency upgrade is authorized.

## Tests
- Stop pinning the full coordinator baseline in tests and goldens: update affected approval, context, blocked-report and force-wire expectations while retaining substantive checks (regression)
- Preserve the complete shaping suite against the current pin and coordinator 482d565 or later: run the mandatory full-suite commands above (integration)
- Additional coordinator automatic paths must not break consumers: run regression acceptance and the full-suite command with the scratch baseline extension (regression)

## Feasibility
- Local probes found the pinned module and project.Baseline in project/baseline.go; its automatic configuration entry is all = append(all, ConfigPath). The pinned full suite passed (probe exit 0, 151.5 s). go mod download resolved 482d565 to v0.0.0-20261103204401-482d5656e610 (probe exit 0). A timing experiment with -parallel=64 took 224.4 s and failed; it is not used. Initial readiness timed out at 120 seconds; full-suite delivery remains required outside the bounded acceptance command. No external protocol documentation is needed for this local consumer-test repair.
- evidence: {"id":"exit-observer","kind":"observer","inputs":["features/steps/k-shape-protect-list-unpinned__steps.js"],"observers":["features/steps/k-shape-protect-list-unpinned__steps.js"],"checks":[{"name":"good","command":"node features/steps/k-shape-protect-list-unpinned__steps.js --control good","exit":0,"output":"accepted\n"},{"name":"bad","command":"node features/steps/k-shape-protect-list-unpinned__steps.js --control bad","exit":1,"output":"rejected\n"}]}
