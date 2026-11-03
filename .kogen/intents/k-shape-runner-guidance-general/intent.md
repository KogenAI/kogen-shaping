# Remove runner-specific runner-observer examples

Slug: k-shape-runner-guidance-general
Model: not applicable — two fixed texts change (one guidance paragraph and one refusal headline); the refusal gate, its order and its state are unchanged
Acceptance:
- .kogen/intents/k-shape-runner-guidance-general/acceptance.feature
- features/steps/k_shape_runner_guidance_steps.js
- features/steps/k_shape_runner_guidance_test.go.txt

K-SHAPE-RUNNER-GUIDANCE-GENERAL. kogen-delivery filed it on 3 Nov 2026 under the owner's binding ruling, via main and the research audit SHAPING-AUDIT-20261103: shaping changes are general mechanisms only, and nothing from outside a task's own spec and app reaches the shaper. Repository: kogen-shaping. change_type: fix.

Two texts that Shape sends to its shaper name one test runner. They give its reporter wording ("N runs" vs "N tests", `CI_PROGRESS_BAR=0`, `bin/rails test`) and its `-n`/`-i` name-filter semantics: knowledge from outside a task's own app. The texts are the test-runner observer guidance paragraph (`runnerObserverGuidance` in `shape/context.go`) and the runner-observer refusal headline (`runnerObserverError.text()` in `shape/shape.go`). Both keep only the general rule. A step that judges a test runner's output must accept that runner's real output on the unchanged tree, in the acceptance environment. Summary wording and name-filter behaviour come from a real run of an existing passing test there.

This change removes:

- the guidance's whole sentence beginning `For example, minitest's default reporter ...` and ending `... never ^name$.`;
- the refusal headline's two parentheticals: ` (minitest's default reporter prints "N runs", other reporters "N tests")` and ` (minitest's -n and -i)`.

Nothing else in either text changes. The new texts, exactly:

- `runnerObserverGuidance` (still the last paragraph of `ShaperInstructions()`, after the unwritten-tests paragraph, and still absent from Refine's instructions):
  `Test-runner observers: a step that runs a test runner and judges its output must accept that runner's real output on the unchanged tree, in the environment acceptance runs in. Derive the summary wording and the name-filter semantics from a real run of an existing passing test there, not from memory; an observer control's known-good input is that real output, not hand-written runner text. Shape refuses a candidate whose step fails although the test runner it ran reported a passing run, or whose anchored name filter made the runner run nothing.` followed by one newline.
- The refusal text: its first line becomes, with `%d` the refused scenario count:
  `Acceptance rejects the test runner's real output: %d scenario(s) fail although the test runner reported a passing run, or pass an anchored regular expression as a name filter the runner matches literally; nothing was published. A step that observes a test runner must accept the summary the runner really prints on the unchanged tree and give a literal name filter only the plain test name. Rewrite these steps; resubmitting them unchanged is refused again.`
  After it, as now, come a newline and the item lines, joined by newlines.

The refusal gate itself does not change. It is `readinessGate`/`runnerObserverGate`, together with the kogen-verification analysis that produces each scenario's `RunnerObserver` item. These stay the same:

- which candidates it refuses: a step that fails although the runner reported a passing run, or an anchored name filter that made the runner run nothing;
- its order after the unwritten-tests refusal;
- its item lines;
- its `submission_refused` event with gate `runner_observer`;
- the three-refusals not-ready end;
- the external Shape feedback;
- Refine's identical refusal text, through `refusal()`.

This change adds no CLI commands or flags, and its tests use synthetic runners.

The policy golden `shape/testdata/shaper_policy_golden.json` records `ShaperInstructions()` byte for byte. This is a deliberate, reviewed policy change, so regenerate it with `go test ./... -update` and inspect the diff. Only the `prompt` string may change, by the removed sentence. `prompts_golden.json` and `defs_golden.json` do not change, because the embedded prompt files are untouched.

## Decisions
- The guidance keeps its existing general sentences and drops only the runner-specific example sentence, including its `-n`/`-i` part (source: request)
- The refusal headline keeps its existing general wording and drops only the two runner-specific parentheticals (source: request)

## Source comparison
- Problem: `shape/context.go` runnerObserverGuidance names `"N runs"`/`"N tests"`, `CI_PROGRESS_BAR`, `bin/rails test` and `-n`/`-i`. Addressed: the example sentence is removed; the new exact text is above. Scenario 1 asserts that the instructions end with it and that none of the runner-specific literals appear.
- Problem: `shape/shape.go` refusal text (lines 1649-1650) carries the same examples. Addressed: both parentheticals are removed; the new exact headline is above. Scenario 2 asserts the exact refusal and that its headline names no runner.
- Need: keep the general rule. Addressed: the general sentences stay verbatim. The request's quoted rule is the meaning of those existing sentences, not new wording.
- Constraint: no behaviour change in the refusal gate. Addressed: scenario 2 asserts the same two items, the `runner_observer` gate with 2 problems on each of two refusals, and publication of the corrected candidate. The existing tests in `shape/runner_observer_test.go` keep covering the not-ready end, the external feedback and the unwritten-wins order.
- Constraint: no new CLI commands or flags. Addressed: none are added.
- Constraint: tests use synthetic runners. Addressed: the acceptance replays failure messages of an invented `./bin/orchard-check` through the scripted acceptance runner. Its summary line has the shape the unchanged gate reads, so the gate still fires.
- Omitted: the gate's detection in kogen-verification (another module) and its comments; the existing synthetic report fixtures inside `shape/runner_observer_test.go`, which only feed the scripted runner; other prompts' examples, such as the named-test helper's `go test -run`. The request does not ask for any of these.

## Tests
- The test-runner observer guidance carries only the general rule, with no runner-specific reporter wording or name-filter example: acceptance scenario 1 (Shape's real first request); `shape/testdata/shaper_policy_golden.json` regenerated with `-update` and checked by the existing TestShaperPolicyGolden; the existing TestNamedFeedbackShapeGuidance unchanged (regression)
- The runner-observer refusal headline names no runner: acceptance scenario 2; update the expected `runnerObserverHeadline` constant in `shape/runner_observer_test.go` to the new exact text (regression)
- The runner-observer refusal gate's behaviour is unchanged: the existing tests in `shape/runner_observer_test.go` (refused twice then published, three refusals end not ready with the items, external Shape feedback, unwritten-tests refusal wins) and the k-refine-acceptance-gates acceptance pass unchanged except for the headline constant (existing)

## Feasibility
- Session evidence: my own scratch runs in a copy of HEAD 1e3f967 outside the checkout, not Kogen probe records. Tools: Go 1.27.2, Node v24.21.0, and Kogen's Cucumber at `$KOGEN_CUCUMBER`.
- On the unchanged tree, both journeys fail on the missing feature only:
  - guidance: the instructions do not end with the new paragraph, and they carry `minitest`, `CI_PROGRESS_BAR`, `bin/rails`, `N (runs|tests)`, `-n and -i`, `^name$` and `1 runs, 7 assertions`;
  - refusal: both refusals carry the old headline. Their two items, the two `runner_observer` refusal events with 2 problems each, the readiness observations of the corrected candidate and its publication already hold.
- In a prototype of this contract (the two text edits, the test constant, and the policy golden regenerated with `-update`), the configured acceptance command passed 2 scenarios (4 steps) in about 5 s. `go vet ./...` passed, and `go test ./shape -run 'Runner|Policy|Prompt|Golden|NamedFeedback|Unwritten|Refine'` passed in about 50 s.
- The harness runs Shape with a scripted provider and a replayed acceptance report, and judges each journey with several assertions. Its judges are qualified by controls that use the same submitted bytes:
  - `good` judges correct guidance and refusal observations and must accept both;
  - `bad` judges 21 known-bad observations, each breaking one rule. Examples: the old paragraph or headline, a dropped or duplicated paragraph, the paragraph not last, the example moved elsewhere, Refine carrying it, a dropped item, another gate, one refusal, an unpublished or wrongly published candidate, a not-ready end. It exits 1 only when every one of them is rejected.
  - Observed: good printed `accepted` with exit 0, and bad printed `rejected` with exit 1.
- evidence: {"id":"runner-guidance-observer","kind":"observer","inputs":["features/steps/k_shape_runner_guidance_steps.js","features/steps/k_shape_runner_guidance_test.go.txt"],"observers":["features/steps/k_shape_runner_guidance_steps.js","features/steps/k_shape_runner_guidance_test.go.txt"],"checks":[{"name":"good","command":"\"$KOGEN_NODE\" features/steps/k_shape_runner_guidance_steps.js good","exit":0,"output":"accepted\n"},{"name":"bad","command":"\"$KOGEN_NODE\" features/steps/k_shape_runner_guidance_steps.js bad","exit":1,"output":"rejected\n"}]}
