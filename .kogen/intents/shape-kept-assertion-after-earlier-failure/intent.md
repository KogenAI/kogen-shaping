# Check kept steps after earlier failures

Slug: shape-kept-assertion-after-earlier-failure
Model: not applicable — this repairs validation of fixed readiness observations, not a new state machine
Acceptance:
- .kogen/intents/shape-kept-assertion-after-earlier-failure/acceptance.feature
- features/steps/shape-kept-assertion-after-earlier-failure__steps.js
- features/steps/shape-kept-assertion-after-earlier-failure__driver/observe_test.go

Refuse a draft before publication/Build when its counted, observable unchanged-behaviour assertion cannot be validated on the unchanged tree because an earlier new-behaviour step masks it. Remove the first-failure-is-an-assertion prerequisite from observing the explicitly named kept_steps. In particular, a new-tool error must not bypass the existing kept-step baseline handling. Preserve the existing two-observation bound and feedback identifying the scenario and kept feature line, explaining that the kept assertion must run on the baseline (for example in its own scenario). Do not automatically reorder, execute skipped steps out of sequence, rewrite or drop the draft's scenarios or bindings.

This is not blanket rejection of missing new behaviour: only a named kept step that is assertion-red or blocked in the existing unchanged-tree observations triggers the added baseline path. A blocked step is unobserved, not proof that its expected value is false; require a baseline-runnable assertion instead of publishing an unverified kept claim. A passing kept step followed by missing new behaviour remains eligible, as does a scenario containing only new behaviour. Keep counted-property filtering and existing confirmed/refuted semantics, including the existing non-overridable baseline refusal for explicitly named kept steps. No new setting, dependency, provider call or change to Studio is requested.

Add a focused production regression through Shape with real Cucumber: a scenario first fails on new behaviour with a non-assertion error and a later, wrong kept assertion never runs. Test both a nested document-envelope mismatch and the supplied test-summary mismatch as local synthetic examples; do not claim these fixtures run Studio or ExUnit. Keep useful existing kept/new, persistent/transient, and contract-verdict tests unchanged. (evidence: shape/kept_baseline.go; shape/kept_baseline_test.go; shape/kept_mixed_test.go; shape/kept_repeat_test.go; shape/contract_check_test.go)

## Decisions
- Trigger the existing bounded kept-step baseline check from named red or blocked kept_steps even when the scenario's first failure is not an assertion. (decision: D1, round 1) (evidence: shape/contract.go; shape/kept_baseline.go; shape/kept_baseline_test.go) (source: request)
- Preserve existing confirmation verdict counting and baseline-refusal semantics; a missing new behaviour alone and a passing kept step are not kept failures. (decision: D2, round 1) (evidence: shape/property.go; shape/kept_mixed_test.go; shape/contract_check_test.go; shape/kept_baseline.go) (source: request)

## Source comparison
- Addressed: the request's first-failure blind spot in shape/contract.go by the named kept-step trigger above and acceptance scenarios rejecting the masked document and summary expectations. The independent harness invokes Shape and real Cucumber, not a source-text predicate.
- Addressed: studio-find-and-link-records (eb8361f8), its scenario 3 new studio_records call at line 45 masking the kept document.body assertion at line 51, and lib/studio/document.ex:133 nesting under latest, by the feature's historical context plus executable local document-envelope fixture. The local fixture uses its own feature line numbers, not historical Studio line numbers.
- Addressed: the reported 5 of 14 reds, persistent kept assertion red in studio-concise-agent-intake and k-docs-concise-task-first, and studio-issue-next-action's summary discrepancy by feature context and a masked summary assertion using the supplied strings. These reports motivate the fix, not a claim of reproduced live Studio history. Studio issue kogen/921054ec-feeb-46cf-8950-f8f3c955d2b1 remains a reference, not an external service dependency.
- Addressed: do not refuse merely unimplemented new behaviour by the new-only and passing-kept-before-new acceptance cases. Only explicit kept_steps evidence drives the additional baseline handling; unchanged observation and refusal semantics are retained rather than inventing an assertion result for a skipped tail.
- Addressed: keep confirmed/refuted semantics by a refuting scripted confirmation in the regression harness and retaining the existing verdict-counting and mixed-property tests. The existing explicitly named kept-step baseline refusal remains independent of reviewer refutation.
- Omitted: no requested behaviour. Conflicting or superseded: only the first-assertion gate that bypasses existing kept_steps handling is removed. Studio fixes, ExUnit parsing changes, provider work, and unrelated recovery work are outside this request.

## Tests
- A new-behaviour non-assertion failure must not mask a later wrong kept document assertion: add a focused Shape/Cucumber regression and run the document scenario (regression)
- The supplied summary discrepancy must not escape behind the same early failure: run the masked summary fixture (regression)
- Explicit blocked kept-step refusal remains effective with a refuted scenario-level finding: run the earlier-assertion fixture and retain TestKeptStepRedOnBaselineIsRefusedDespiteRefutation (integration)
- Missing new behaviour without a failing or blocked kept assertion is not refused by the kept check: run new-only and passing-kept fixtures (integration)
- Existing confirmed/refuted, mixed-property and repeat semantics remain unchanged: retain the contract, kept baseline, mixed and repeat test suites (existing)

## Feasibility
- Existing TestKeptStepRedOnBaselineIsRefusedDespiteRefutation passed with real local Cucumber. (probe: go test ./shape -run '^TestKeptStepRedOnBaselineIsRefusedDespiteRefutation$' -count=1; exit 0, 7.1 s including tool discovery)
- An initial synthetic constant assertion was rejected by the existing unreachable-step gate; the final fixture instead reads the local module's prospective studio_records result, so a correct implementation can satisfy it. The final observer controls below reached the existing blocked-kept refusal, independently of the new non-assertion case.
- evidence: {"id":"masked-kept-observer","kind":"observer","inputs":["features/steps/shape-kept-assertion-after-earlier-failure__steps.js","features/steps/shape-kept-assertion-after-earlier-failure__driver/observe_test.go"],"observers":["features/steps/shape-kept-assertion-after-earlier-failure__steps.js"],"checks":[{"name":"good","command":"node features/steps/shape-kept-assertion-after-earlier-failure__steps.js good","exit":0,"output":"observer accepted\n"},{"name":"bad","command":"node features/steps/shape-kept-assertion-after-earlier-failure__steps.js bad","exit":1,"output":"observer rejected\n"}]}
