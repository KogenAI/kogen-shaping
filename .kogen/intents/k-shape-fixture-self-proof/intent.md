# Refuse acceptance whose preparation fails

Slug: k-shape-fixture-self-proof
Model: not applicable — one judgement of each readiness result plus an append-only list of the failure signatures this Shape run already refused; no interacting state transitions
Acceptance:
- .kogen/intents/k-shape-fixture-self-proof/acceptance.feature
- features/steps/k-shape-fixture-self-proof__steps.js
- features/steps/k-shape-fixture-self-proof__harness.js

Today Shape's readiness check (`certify` in `shape/shape.go`, calling kogen-verification's `verify.Readiness.Check`) runs the candidate's acceptance once on the unchanged tree in a fresh scratch copy. It counts every failed scenario as red to acknowledge, whichever step failed. A Given step whose fixture its own consumer rejects (the request's `fixturecheck.js` exiting 2 with `missing kind=packet`) comes back as observations. Acknowledging that unchanged package publishes it. The contract review reports such a defect only when its model finds one.

After this change, Shape's readiness gate (`sh.rd.Gate` in `shape/shape.go`) adds a preparation gate between `readinessGate` (unwritten tests, then runner observer) and `unpassableGate`. The gate judges the `*verify.Ready` of the same authoritative readiness run: the same fresh copy and the same bounds. It adds no run, changes no timeout, runs no step out of sequence and uses no list of repository names, configuration keys or error strings. kogen-verification is pinned and is not edited. Its `ScenarioResult` already holds `Steps`, `Status`, `Detail`, `UnexercisedSteps`, `OutlineLine`, and `Ready` holds `AcceptanceExit`.

Rules, for each scenario with `Status` `failed` and a non-nil `Detail`:

1. The first line of `Detail` (`<where>`) locates the failure. Either it is `acceptance.feature line <L>: ...`, which names the step of `Steps` with line L, or it starts with `a hook`.
2. The failure is a preparation failure, with the reason `<why>`, when:
   - the failed step's line is lower than the scenario's own line (`*OutlineLine` for an Examples row, otherwise `Line`), so it is a Background step: `a Background step`;
   - otherwise, the failed step's keyword is `Given `, or it is `And `, `But ` or `* ` and the nearest earlier entry of `Steps` with keyword `Given `, `When ` or `Then ` is `Given `: `a Given step`;
   - a hook failed and every step of the scenario is in `UnexercisedSteps`, so it failed before the first step ran: `a Before hook`;
   - otherwise (a When or Then step, or a conjunction continuing one), its signature equals the signature of a preparation failure that this Shape run refused at an earlier submission: `the same failure as the preparation refused at attempt <n>`, where n is the earliest such attempt. This is how a broken builder moved from a Given into a When or Then stays refused.
3. The signature is `Detail` without its first line, redacted, with every line removed whose text after leading spaces and tabs starts with `at ` (stack frames, as `withoutFrames` in `shape/unpassable.go` does), and with trailing whitespace trimmed. An empty signature never matches.
4. These are not preparation failures:
   - a When or Then failure with no earlier matching signature;
   - a hook failure after a step ran, such as an After hook;
   - any scenario that did not fail.
   A skipped step is never judged: it is neither a failure nor a proof.
5. If at least one scenario has a preparation failure, the gate refuses:
   - `Check` returns the gate's error, so no readiness record is written;
   - the unpassable control run and the contract review do not run for that candidate;
   - `submission_refused` is recorded with gate `preparation` and the item lines as problems;
   - each refused failure's signature is remembered for the rest of this Shape run, with its attempt number (`sh.submissions`).
   The shaper receives this text:
   - The headline, one line, where N is the number of scenarios with a preparation failure: `Acceptance preparation failed on the unchanged tree: <N> scenario(s) fail before they reach the behaviour under test, so this is not missing-feature red; nothing was published. A failed Given or Background step or Before hook, or a When or Then step that fails exactly as a refused preparation did, is a defect of the acceptance package: repair the preparation so that it succeeds on the unchanged tree and the consumer of its fixture accepts it. Keep the scenario and its outcome assertions, and do not change production code to accept a malformed fixture. Resubmitting or acknowledging this package unchanged is refused again.`
   - Then, for each such scenario in readiness order:
     - `- <Name> (acceptance.feature line <Line>): <where> failed (<why>); acceptance exit <AcceptanceExit>`
     - `  <line>` for each line of the output. The output is `Detail` without its first line, redacted with `verify.NewRedactor(session env, HOME)` and then bounded with `verify.Diagnostic`; an empty line stays empty. Its first line is the failing operation as the runner reported it, for example `AssertionError [ERR_ASSERTION]: fixturecheck exit=2`.
     - `  not run: acceptance.feature line <L>: <Keyword without its trailing space> <Text>` for each entry of `UnexercisedSteps`, in order.

   For the request's package the item lines read:
   - `- Count a prepared packet (acceptance.feature line 2): acceptance.feature line 3: Given fixture-proof: a packet failed (a Given step); acceptance exit 1`
   - then `  AssertionError [ERR_ASSERTION]: fixturecheck exit=2` and `  missing kind=packet`
   - and finally `  not run:` lines for lines 4 and 5.
6. Readiness runs again for every submission, as today, so an unchanged acknowledgment is refused again. The existing `Unchanged: this package is identical to the one refused at attempt <m>; repair it before submitting again.` line is appended to it as today. After the retry budget, the existing `intent/not_ready` stop lists the item lines.
7. Red that is not a preparation failure keeps today's route: unpassable gate, contract review, observations, acknowledgment and publication.

Boundary: the gate cannot tell from code alone what a When or Then step reaches. A builder that fails inside a When or Then step is refused by this gate only if the same failure was refused earlier in this Shape run as a Given or Background step or a Before hook. Otherwise the existing advisory contract review still applies. The signature list lives only for one Shape run. External environment failures stay the honest stops they are today, with their original output: acceptance unavailable, acceptance timeout, setup and model failures, provider or authentication errors. Nothing invents a passing fixture.

Unchanged:
- Refine, which keeps its own `readinessGate` and `unpassableGate`;
- the step self-check, contract review, unwritten-tests, runner-observer and unpassable checks and their texts;
- every prompt, tool definition and policy golden (`shaper_policy_golden.json`, `prompts_golden.json`, `defs_golden.json`), CLI flag and README.

Test fixture: `shpMessages` in `shape/helper_test.go` fails the first step of every `FAILED` scenario today, and that step is the Given of `shpFeature`. With this change that replay is a preparation failure. So `FAILED` now marks each scenario's last step failed, with the same message, and every earlier step passed. Tests that asserted the old report's skipped steps or its Given failure location are updated to the new report. Examples are `TestShapeRedReadinessReturnsObservationsAndAcknowledgmentReusesThem` and the Refine observation tests in `shape/refine_test.go` that expect `Unexercised acceptance steps: 2` or `acceptance.feature line 3: Given is-even: the number 4`. Then regenerate only the force goldens `shape/testdata/force_ordinary_golden.json` and `shape/testdata/force_ordinary_models_golden.json` with `go test ./shape -update`. Inspect the diff: only the hashes of request bodies that carry readiness observations may change. The acceptance harness compiles against these test helpers, so keep them: `newShpFxRun`, `shpSubmit`, `shpOutputs`, `shpStep`, the `shpFx` fields `Root`, `Aux`, `Env`, `S`, `Client`, `Prov`, the methods `write`, `git` and `newSession`, and `Prov.requests`.

## Decisions
- A failed setup step (a Given or Background step or a Before hook) on the unchanged tree is a shaping defect. It is refused before publication even when the contract review returns no finding, and an unchanged acknowledgment cannot waive it. (source: request)
- The refusal returns the scenario, the failed step's location, the failing operation, the acceptance exit status, the redacted and bounded output, and the steps that did not run. (source: request)
- A broken builder that is moved from a Given into a When or Then step and fails the same way is still refused. (source: request)
- Outcome red that reaches the behaviour under test keeps the ordinary acknowledgment and publication route. A deliberately invalid input whose construction succeeds is a valid fixture. (source: request)
- The gate reuses the authoritative readiness run and its bounds: no second whole-suite run, no raised timeout, no skipped step run out of sequence, and no claim that an unexecuted fixture passed. (source: request)

## Source comparison
- Addressed: request Need. Rules 1–7 add an enforced preparation gate to Shape's readiness gate, judged from the same fresh-copy readiness run. The gate runs before the unpassable control and the contract review, so a no-finding review cannot let a preparation failure through. Feedback holds the scenario, step location, failing operation (first output line), exit status (`acceptance exit`), redacted and bounded output, and the not-run steps.
- Addressed: "do not add a list of special repository names, configuration keys or error strings". Classification uses only the Gherkin role of the failed step (Given, a conjunction continuing a Given, a Background step), the hook-before-first-step shape of the result, and this run's earlier refused failure signatures. The consumer's own exit and message (`fixturecheck.js`) show fixture validity; Kogen does not interpret them.
- Addressed: request Synthetic acceptance, with its exact `count.js`, `fixturecheck.js`, feature and `features/steps/fixture-proof__steps.js` literals in the harness (a synthetic Intent with slug `fixture-proof` owns that step path). Every contract review answers no finding.
  - Item 1: the "missing-kind" scenario.
  - Item 2: the "repair" scenario. The unchanged acknowledgment is refused. The repair changes only `this.packet` to `{ kind: 'packet', items: ['a', 'b'] }`; then Given and When pass, the Then fails with `0 !== 2`, and the package is acknowledged and published.
  - Item 3: the "items-not-array" scenario, with `{ kind: 'packet', items: 0 }` and `items must be an array`.
  - Item 4: the "negative-input" scenario. Its Then asserts exit 2 and exactly `items must be an array\n`, and it is published with no refusal.
  - Item 5: the "moved-builder" scenario. The same builder at the start of the When body, followed by the count call, is refused as the same failure as attempt 1. The "skipped-builder" scenario shows that a builder skipped behind legitimate outcome red is not executed, not refused, and stays listed as unexercised.
- Partly addressed: "Step role is semantic". A failing builder moved from Given into When or Then is refused through its signature. A builder written first inside a When or Then, never refused as preparation, is not detected by this gate: only the advisory contract review applies there (Boundary). Telling such a builder from legitimate outcome red would need knowledge of the behaviour under test that a readiness result does not hold.
- Omitted: Refine. The request names Shape and the shaper; Refine's gates are unchanged.
- Superseded: the replayed test report's default red at the first (Given) step, which is now a preparation failure. It moves to each scenario's last step, and the force goldens are regenerated for that.

## Tests
- A failed Given step, or a conjunction continuing a Given, is a preparation failure with the exact item, output and not-run lines: add `TestPreparationFailures` in `shape/preparation_test.go` over `verify.Ready` values, with cases for a Given, an `And` after a Given, an `And` after a When (not preparation) and a passed scenario (nothing) (unit)
- Background steps (by line, including an Examples row through `OutlineLine`) and a hook failing before the first step are preparation failures; a hook failing after the steps ran is not: cases in `TestPreparationFailures` (unit)
- A When or Then failure whose signature equals an earlier refused preparation failure, with different stack frames, is refused and names the earliest attempt; one with a different message, or an empty signature, is not: cases in `TestPreparationFailures` (unit)
- The output is redacted and bounded: a case whose message holds `api_key=abc123` shows `api_key=[REDACTED]` (unit)
- Through Shape with the replayed runner, a report whose Given fails is refused at gate `preparation`. It has the headline, no readiness record, no contract review request and no `unpassable_check` event. Resubmitting it unchanged is refused again with the Unchanged line: add `TestShapePreparationGate` in `shape/preparation_test.go` (integration)
- The request's five synthetic journeys and the skipped-builder control end to end with real Node and Cucumber: the scenarios in acceptance.feature (scenario)
- Legitimate red still returns observations and is published after acknowledgment: `TestShapeRedReadinessReturnsObservationsAndAcknowledgmentReusesThem`, updated to the last-step report (existing)
- Unwritten-tests and runner-observer refusals still win, and Refine is unchanged: their existing tests, updated only for the report's failure location (existing)

## Feasibility
- A scratch copy of the unchanged tree, outside the checkout, ran this acceptance with the configured runner, Kogen's Node 24.21.0 and Cucumber: 6 scenarios, 2 passed (negative-input, skipped-builder) and 4 failed at their first `submission 1 is refused at preparation`, because today Shape returns accounted red observations. That is missing-feature red. (probe: exit 1, 17 s)
  - Observed in real readiness: the Given fails with `acceptance.feature line 3: Given fixture-proof: a packet` and `AssertionError [ERR_ASSERTION]: fixturecheck exit=2` / `missing kind=packet`, with the When and Then unexercised and acceptance exit 1.
  - The moved builder fails in the When with the identical message except its stack frame (`:17:10` instead of `:15:10`).
  - The skipped-builder package returns `0 !== 2` at line 5 with line 6 unexercised, its body never ran, and it was published after acknowledgment.
  - The embedded literals are byte-identical to the request's.
- evidence: {"id":"fixture-proof-observer","kind":"observer","inputs":["features/steps/k-shape-fixture-self-proof__steps.js","features/steps/k-shape-fixture-self-proof__harness.js"],"observers":["features/steps/k-shape-fixture-self-proof__steps.js"],"checks":[{"name":"good","command":"node features/steps/k-shape-fixture-self-proof__steps.js good","exit":0,"output":"accepted\n"},{"name":"bad","command":"node features/steps/k-shape-fixture-self-proof__steps.js bad","exit":1,"output":"rejected\n"}]}
