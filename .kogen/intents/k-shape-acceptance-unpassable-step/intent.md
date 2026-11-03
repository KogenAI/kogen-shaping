# Refuse acceptance steps no implementation can pass

Slug: k-shape-acceptance-unpassable-step
Model: not applicable — one stateless judgment of a candidate's readiness result plus one control run; refusals use the existing bounded correction loop and add no workflow state
Acceptance:
- .kogen/intents/k-shape-acceptance-unpassable-step/acceptance.feature
- features/steps/k_unpassable_step_steps.js
- features/steps/k_unpassable_step_test.go.txt

`kogen intent shape` must refuse a candidate whose acceptance has a step definition for which no pass is reachable: it fails for every implementation and every repository state. The refusal names the step and says why, and goes back to the shaper as a repairable problem. Today such a step, for example a stub that throws "coverage incomplete", counts at readiness as ordinary missing-feature red. The contract check (shape/contract.go) cannot report it either, because there is no Intent or code quote for it to contradict. So the candidate is published, and the Build stops on a baseline contradiction. Steps that fail on the unchanged tree only because the requested feature is missing must stay approved exactly as today: same observations and acknowledgment text, same publication. The check is model-free. It never edits, deletes or weakens a scenario, replaces no existing gate, and does not depend on the example-consistency gate (d1ffe29). Only Shape changes: `kogen intent refine`, `kogen intent approve`, the step self-check, the contract check and kogen-verification readiness are unchanged.

Where it runs: in Shape's readiness gate (`readinessGate`, installed as `sh.rd.Gate` in `(*shaping).shape`, shape/shape.go). It runs after the unwritten-tests and runner-observer refusals, which keep their precedence. It runs only when the readiness result has at least one failed scenario, and only before any readiness evidence is recorded. It judges every step definition that is the failing step of a failed scenario (the location in the readiness failure detail, `acceptance.feature line <L>: <Keyword> <text>`, maps to its definition).

Rule (a property of the step definition). A definition is unpassable ("no pass is reachable") only when all of these hold:
1. On the readiness run it is the failing step of at least one failed scenario, and it passed in none. A step passed when its scenario passed, or when it came before the failing step of a failed scenario; each Examples row of an outline counts as its own scenario.
2. None of those failures is a timeout (Cucumber's `function timed out, ensure the promise resolves within N milliseconds`). A step that times out, for example while waiting on a service the sandbox does not provide, is an environment problem and never unpassable. Its outcome stays exactly today's: accounted red returned with the existing acknowledgment text, or the existing `acceptance timed out` refusal.
3. In Kogen's control run (below), each of those scenarios fails at the same step with the same failure, and no stand-in was used in that scenario up to and including that step. "Same failure" means the readiness failure detail and the control detail are equal once stack-frame lines (lines that start with `at ` after leading whitespace) are removed.

Anything else is reachable and is not refused.

The control run:
- What runs: Kogen's own Node and Cucumber, resolved the same way as the step self-check (the readiness tools, else `KOGEN_NODE` / `KOGEN_CUCUMBER`). It runs once per checked candidate, on the candidate's acceptance.feature, with a Cucumber Messages report.
- Where: a fresh scratch directory materialized from the readiness result's `CandidateTree`, with no setup commands run. The run is bounded by `verify.StepTimeout`, and the directory is removed afterwards.
- What loads: the same step files as the step self-check (`verify.StepAnalysis`), after a Kogen-supplied preload and support hooks embedded in the Go binary. The preload loads before any step file (for example `node -r`).
- What gets interposed: everything acceptance code obtains from outside itself. Acceptance code means the JavaScript files under `features/steps/`.
  - Modules it loads (require or import), other than Node builtins, `@cucumber/cucumber` and files under `features/steps/`, become stand-ins. This includes modules that cannot be resolved.
  - Results of file-system reads, child_process functions and network calls (net, tls, http, https, dgram, fetch) made directly by acceptance code become stand-ins. No process is started and no connection is made.
  - Calls whose result cannot be a stand-in count as a use when called: `fs.existsSync`, `fs.accessSync` and asynchronous file-system calls. Reading `process.env` from acceptance code also counts as a use.
- What a stand-in is: a value on which every operation by acceptance code counts as a use and fails the step. Operations include property access, call, construction, `in`, key listing, conversion to a primitive and awaiting it. Operations by other code, such as Node's assert internals or Cucumber's formatters, are inert and never count, so a stand-in can never break Cucumber's own reporting. (A prototype crashed the whole run until this rule was added; see Feasibility.)
- Fail-open: the control refuses only on positive evidence. Suppose Node or Cucumber is unavailable, a step file does not load, the report is missing or invalid, the bound is exceeded, or a failed scenario or its step cannot be matched in the control report. Then nothing is refused on that basis, and the event records `skipped` with the reason.

Verdicts, one per judged definition:
- `timeout`: a failing use timed out.
- `passed`: a use passed on the readiness run.
- `stand_in_used`: in the control run a stand-in was used in that scenario before or at the failing step.
- `different_failure`: the control run passed that step, failed at another step, or failed with a different failure.
- `unpassable`: rules 1 to 3 hold.

When a definition has several failing uses, the reported verdict is the first of these, in this order, that applies to any use: timeout, passed, stand_in_used, different_failure. It is unpassable only when none applies.

Event: each check records exactly one shaping event of this shape: `{"event":"unpassable_check","ok":<true unless a definition is unpassable>,"steps":[{"step":"acceptance.feature line <L>: <Keyword> <text>","definition":"<file> line <n>","pattern":"<pattern source>","verdict":"<verdict>"}]}`. `steps` has one entry per judged definition, ordered by its first failing scenario. `step` is that first failing use, exactly the first line of its readiness failure detail; for an outline row it is the outline step's line with the row's text, for example `acceptance.feature line 3: Then 3 is even`. A skipped control records `{"event":"unpassable_check","ok":true,"skipped":"<reason>","steps":[]}`.

Refusal:
- What happens: when a definition is unpassable, the gate returns an error (like `unwrittenTestsError`). Nothing is recorded or published. `checkSubmission` records `submission_refused` with gate `unpassable_step`, with the item lines as its problems. It returns the text below to the shaper as an invalid submission, which counts against the existing correction allowance.
- At exhaustion: Shape ends with `intent/not_ready` "the Intent is not ready to build", with the item lines.
- The text, exactly:

```
Acceptance has steps that no implementation can pass: <N> step definition(s) fail the same way whatever the repository contains, so no pass is reachable; nothing was published. Each step must be able to pass once the requested behaviour exists: run the program, call the code or inspect its output, and compare what it observed with an assertion that shows the actual value (for example assert.strictEqual). Rewrite these steps; resubmitting them unchanged is refused again.
- acceptance.feature line <L>: <Keyword> <text> (step definition "<pattern>", <file> line <n>): no pass is reachable: it fails with "<first message line>" whatever the repository contains
```

- The fields: there is one item per unpassable definition, at its first failing use. `<first message line>` is the first line of the readiness failure message (the detail line after the location), cut to 200 characters. Definitions with any other verdict are never listed.

Unchanged:
- The text of every outcome that is not this refusal: the acknowledgment and observations, and every existing refusal, including the acceptance-timeout refusal.
- The provider requests of ordinary sessions. The force golden (shape/testdata/force_ordinary_golden.json) must stay byte-identical; do not regenerate it.

Build notes:
- The acceptance harness compiles `shape/k_unpassable_step_acceptance_test.go` from `features/steps/k_unpassable_step_test.go.txt` through a Go overlay. Do not create that path, and do not use the `kunp` or `KUnpassable` identifier prefixes in the package.
- Keep the shaping test helpers the harness calls working with their current signatures: `newShpFx`, `shpFiles`, `shpIntentMD`, `shpSubmit`, `shpStep`, `shpOutputs`, `shpNamed`, the `shpFx` methods `write`, `read`, `exists`, `git`, `events` and `shape`, and `Prov.seq` and `Prov.requests`.
- A natural home is a new `shape/unpassable.go` with its tests in `shape/unpassable_test.go`.

## Decisions
- Refuse such a candidate at shape time, name the step with its feature line and definition, give the reason "no pass is reachable", and do not publish it. (source: request)
- Keep approving, with today's texts, steps that fail only because the feature is missing, steps that pass for another input value, and steps that time out on an unavailable service (an environment problem). (source: request)
- Return the refusal to the shaper as a repairable problem in the existing correction loop, so a repaired submission is checked and accepted on the next turn. (source: request)

## Source comparison
- Addressed: the request's Need, a step is acceptable only when a pass is reachable. The chosen form combines the request's "reachable-pass analysis" (the failure is the same on the unchanged tree and on a perturbed tree) with its "same failure across Kogen's own runs". The perturbed tree is the control run, where everything acceptance code reads from outside itself is replaced by a stand-in whose use is recorded. That makes it model-free, and it never edits a scenario.
- Addressed: synthetic acceptance items 1-7, in acceptance.feature modes `stub` (1), `assertfalse` (2), `missing` (3, which also checks that the step passes once a trivial isEven stand-in exists), `inputs` (4, an outline whose rows 4 and 3 pass and fail), `only` (5, with the default fake contract review answering no findings), `service` (6, a local service that accepts and never answers, so the step's 500 ms Cucumber timeout fires) and `repair` (7). The journeys use a small synthetic repository, a scripted shaper and a real Cucumber runner; no AoR task and no task wording.
- Addressed: the counter-case from shape/contract.go line 47 (a scenario that fails only because the feature is missing is not a finding) is preserved by the `missing`, `inputs` and `service` journeys. A file the Build has yet to create cannot produce an unpassable verdict, because reads return a stand-in instead of ENOENT (seen in the prototype).
- Interpretation: "Approval" in the request means Shape's acceptance and publication of a ready candidate, which is where the problem arises and where every synthetic case runs. `kogen intent approve` and Refine are unchanged. The shaper's own self-check gets the refusal as a correction before its last submission.
- Superseded or not relied on: d1ffe29 (example-consistency gate, gate `example_check`) is not in this checkout. HEAD is f93c22b; shape/examples_revert_test.go only checks that no example check runs. Nothing here overlaps it or depends on it.
- Out of scope (request): benchmark evidence and the held-out screen stay with research.
- Known limit, not claimed exhaustive: a step that compares an outside value only by identity (`===` or `!==`) and then fails with a fixed message shows no stand-in use, so it can be refused even when the feature is merely missing. The refusal text tells the shaper to assert with a call that shows the actual value, which the control does tell apart.

## Tests
- A step that throws unconditionally, in an otherwise valid candidate, is refused with its feature line, definition and "no pass is reachable", is not published, and is reported at exhaustion: the `stub` journey, plus unit tests of the judgment from readiness results and control observations in shape/unpassable_test.go (integration)
- A step that asserts false after reading a file is refused because its failure does not depend on the tree: the `assertfalse` journey, plus an integration test of the embedded preload with real Node and Cucumber covering the stand-in rules (read unused, read used, repository module used, missing file, existsSync, operations by non-acceptance code are inert) (integration)
- A step that fails because a function it calls is missing stays approved (`stand_in_used`) and passes once a trivial stand-in implementation exists: the `missing` journey (regression)
- A step definition that fails for one input value and passes for another stays approved (`passed`): the `inputs` outline journey (regression)
- A candidate whose only step is a stub is refused even though the contract check finds nothing: the `only` journey (integration)
- A step that times out waiting on an unavailable service is an environment problem (`timeout`), and the shaper receives the unchanged readiness acknowledgment: the `service` journey (regression)
- The refusal is repairable: a corrected submission with a real step is checked and published on the next turn: the `repair` journey (integration)
- An unusable control run (Node or Cucumber missing, a step file that does not load, a missing report, the bound exceeded) refuses nothing and records `skipped`: unit tests in shape/unpassable_test.go (unit)
- The unwritten-tests and runner-observer refusals, the acknowledgment text, the contract check and the force golden stay unchanged: the existing shape tests pass unchanged (existing)

## Feasibility
- Session prototype (my own scratch run outside the checkout, not a Kogen probe record), with Node v24.21.0 and the repository's Cucumber 13.3.0 (testdata/tools/cucumber). Cucumber's real failure messages were `Error: coverage incomplete` (then stack frames), `AssertionError [ERR_ASSERTION]: The expression evaluated to a falsy value:` for `assert.ok(false)`, and `Error: function timed out, ensure the promise resolves within 300 milliseconds`.
- With a `node -r` preload giving stand-ins, plus Before/AfterStep support hooks counting uses, the results were:
  - The stub and the read-then-assert-false steps failed identically with zero uses.
  - The `require('../../calc').isEven` step used a stand-in.
  - A read of a not-yet-created file failed differently.
  - A definition that passed for 4 and failed for 3 used nothing.
  - The 300 ms service wait timed out.
- A first prototype in which any operation on a stand-in counted crashed the whole Cucumber run, inside assertion-error-formatter. Making operations by non-acceptance code inert fixed it; hence that rule.
- Unchanged-tree run of the submitted harness, from a scratch copy of HEAD: all 7 scenarios fail on the missing gate (acknowledgment instead of refusal, no `unpassable_check` event), in about 13 s including the one-time overlay compile (a warm `go test -c ./shape` takes about 3 s). The observer controls printed `accepted` (exit 0) for good and `rejected` (exit 1) for bad. The bad run failed on the stub's verdict assertion.
- evidence: {"id":"unpassable-observer","kind":"observer","inputs":["features/steps/k_unpassable_step_steps.js","features/steps/k_unpassable_step_test.go.txt"],"observers":["features/steps/k_unpassable_step_steps.js","features/steps/k_unpassable_step_test.go.txt"],"checks":[{"name":"good","command":"node features/steps/k_unpassable_step_steps.js good","exit":0,"output":"accepted\n"},{"name":"bad","command":"node features/steps/k_unpassable_step_steps.js bad","exit":1,"output":"rejected\n"}]}
