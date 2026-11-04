# Check acceptance against wrong implementations

Slug: k-shape-wrong-implementation
Model: not applicable — one bounded judgement per newly checked candidate after the existing contract check: a fixed sequence of at most 8 runs under one deadline, whose refusal reuses that check's refusal record, package-hash memory and correction loop, with no new workflow state
Acceptance:
- .kogen/intents/k-shape-wrong-implementation/acceptance.feature
- features/steps/k-shape-wrong-implementation__steps.js
- features/steps/k-shape-wrong-implementation__harness.js

K-SHAPE-WRONG-IMPLEMENTATION in kogen-shaping, the second of two items, after K-SHAPE-PROPERTY-COVERAGE (`shape/property.go`). Acceptance can exercise every requested property and be red on the unchanged tree for the right reason, and still pass an implementation that gets the property wrong. A behaviour that must be kept already passes on the unchanged tree, so readiness red cannot show that acceptance would notice a break. A bound or comparison (no slower than, no larger than, same as before) can be asserted so loosely that a clearly worse implementation passes. Readiness runs the candidate's acceptance only on the unchanged tree, and the unpassable-step check (`shape/unpassable.go`) runs it only against stand-ins. Nothing runs it against a wrong implementation.

After this change, `kogen intent shape` checks one more thing before it publishes an Intent whose request asks for properties a step can observe. Kogen runs the candidate's own acceptance with implementations that the shaper supplies in the candidate's `.kogen/intents/<slug>/controls.json`. It runs once with the correct implementation, which must pass every scenario of those properties. It runs once per wrong implementation, and for each property at least one wrong implementation must fail one of its scenarios at an assertion about that property. Otherwise the candidate is refused like the contract check's other refusals:
- nothing is published;
- the refusal names each property and what to repair, or the cap that ran out;
- an unchanged resubmission is refused again;
- when the correction allowance runs out, Shape ends with `intent/not_ready` "the Intent is not ready to build".

This adds to the existing checks and replaces none. Only Shape changes. These stay unchanged: Refine (a separate item; its own call of `contractCheck` in `shape/refine.go` must not run this check), `kogen intent approve`, readiness, the step self-check, the unwritten-tests, runner-observer, preparation and unpassable-step refusals, the contract check's contradiction, environment and property findings, and kogen-verification. The Shaper's prompt, policy and tool definitions do not change, so shape/testdata/shaper_policy_golden.json, prompts_golden.json, defs_golden.json and the force golden stay byte-identical. Do not regenerate them. The shaper learns the form of controls.json from the refusal text.

Where it runs: in `checkSubmission` (`shape/shape.go`), in the branch that contract-checks a newly checked candidate (`sh.lastFresh && !sh.contractChecked[hash]`), right after `contractCheck` returned no confirmed contradiction, environment finding or property problem, and before that branch marks the hash checked.
- The checked properties are the properties of that check's review (`submit_contract_review`) that count (`propertyCounts`), are observable and have at least one declared scenario (`declaredScenarios`), in the review's order. A property's scenarios are its declared scenarios. When a declared name is an outline, every example of the outline is a scenario of the property.
- Without a checked property nothing is read, run or recorded, and the route, outcome, acceptance runs and model requests stay exactly as today. This covers a plain input/output request whose review answers `"properties": []`, a request whose only properties need human evaluation, and an incomplete review.
- The check never makes a model request.
- It runs at most once per package hash. An unchanged resubmission gets the same outcome without new runs. A changed candidate (any changed file of the package, controls.json included, since the file belongs to the package hash) gets fresh runs.

controls.json: shaping may now also write `.kogen/intents/<slug>/controls.json`. `validateFiles` accepts that path. Its message for other paths becomes `<path>: shaping may write only .kogen/intents/<slug>/{intent.md,acceptance.feature,model.qnt,controls.json} and step files under <steps dir>/`; update the expected strings in shape/shape_test.go and shape/propose_files_test.go. The file needs no Acceptance: entry: like every file of the Intent directory, it is part of the package hash and the protected baseline. Kogen reads it only when the check runs. Its form is exactly:

```json
{
  "correct": {"files": {"calc.js": "<whole content>"}},
  "wrong": [
    {"name": "add removed", "scenario": "Add keeps working", "step": "Then is-even: add of 2 and 3 is 5", "files": {"calc.js": "<whole content>"}}
  ]
}
```

- `correct` is an object with exactly the field `files`. `wrong` is a list, possibly empty, of objects with exactly the fields `name`, `scenario`, `step` and `files`.
- `files` maps at least one path to its whole new content (a string). In a run, each such file is written over the copy, created when absent.
- `name` is not blank, and no two names are the same after whitespace normalization (`normalizeWhitespace`).
- `scenario` is the name of a scenario or outline of the candidate acceptance.feature, compared whitespace-normalized.
- `step` is one step line of that scenario as acceptance.feature writes it, without its indentation: keyword and text, compared whitespace-normalized. It names the step that must reject this wrong implementation.
- A path must be relative and normal (`normalPath`). It must not be acceptance machinery:
  - a path under `.kogen/` or under the configured steps directory;
  - a path listed under the Intent's Acceptance;
  - a path a configured `protect` pattern matches;
  - a path with a `.git` or `node_modules` segment;
  - `cucumber.js`, `cucumber.cjs`, `cucumber.mjs`, `cucumber.json`, `cucumber.yaml` or `cucumber.yml` at the root.
- Each implementation, the correct one and each wrong one, has at most 8 files and at most 65536 bytes of content in total.

Runs: the check needs one run for the correct implementation and one per wrong implementation.
- Run cap: at most 8 runs per candidate. A candidate that needs more is refused before anything runs.
- Order: otherwise the correct implementation runs first, then the wrong ones in file order.
- The copy: each run materializes the tree readiness checked (`sh.ready.CandidateTree`, as `unpassableControl` does) into a fresh directory outside the checkout. It writes the correct implementation's files over that copy, and in a wrong run then that wrong implementation's files over them.
- What runs: the configured setup commands in order, as readiness runs them without reuse, then the configured acceptance command in the copy. The command gets `KOGEN_ACCEPTANCE_FEATURES` set to the Intent's acceptance.feature and `KOGEN_ACCEPTANCE_REPORT` set to a Cucumber Messages report path outside the copy. These are the same acceptance, setup and step files the Build is verified with.
- Kogen's own preload: every run also gets `NODE_OPTIONS` beginning with `--require <path>`, any existing `NODE_OPTIONS` kept after it. The preload file is embedded in the shape package and written outside the copy. Before any step file or implementation code loads, it freezes (`Object.freeze`) the module objects of `node:assert` and `node:assert/strict`, every function-valued property of each, and those functions' prototypes. An assignment such as `require('node:assert/strict').strictEqual = () => {}` then changes nothing. The copy and the preload are removed after the run.
- The checkout, its index and HEAD are never written.
- Time cap: the runs of one candidate, setup included, have a total wall-time cap of 300 seconds. `ShapeInput` gets the field `WrongImplementationTimeCap time.Duration` ("internal total-time cap of the wrong-implementation runs per candidate; nonpositive means 300 seconds"), an internal knob like `SubmissionRetryLimit`. Each run is bounded by the time that remains and its process group is killed when that time runs out. A run that is cut off, or that would start with no time left, ends the check with the time-cap refusal, and no further run starts.
- A run is complete when its setup succeeded and its report accounts every scenario of the feature (each example of an outline) exactly once, as passed or failed.
- Reachability: when the correct run is incomplete, or a scenario of a checked property did not pass in it, there is no reachable pass and no wrong implementation runs.

What counts as evidence: for each step Kogen reads its status and, for a failed step, its exception `type` and `stackTrace` from the run's report. Cucumber 13.3.0 writes them under `testStepFinished.testStepResult.exception`; without `stackTrace`, use `message`. A wrong run W is evidence for checked property P when all of these hold:
1. W's run is complete.
2. W's scenario is a scenario of P, and it failed in W's run.
3. Its first step that did not pass is at the line of W's `step` (all earlier steps of it passed).
4. That step's exception `type` is `AssertionError`.
5. The first line of the stack trace that starts with `at ` after its leading whitespace names a file, relative to the copy, that is one of the step files the Intent lists under Acceptance. Strip the copy's path, as given and with symlinks resolved, and a `file://` prefix.
6. Every scenario of the other checked properties that is not a scenario of P passed in W's run.

A failure through a load error, missing file, timeout or unconditional throw is not evidence (rules 4 and 6). Neither is an `AssertionError` that implementation code throws (rule 5), nor a failure at an earlier or unrelated step of the scenario (rule 3). P is satisfied when at least one wrong implementation is evidence for it.

Refusal: the problems are item lines in this order, and each group ends the list.
1. controls.json problems. Nothing runs.
2. The run-cap line. Nothing runs.
3. No reachable pass: one line per checked property with a scenario that did not pass in the correct run. No wrong implementation runs.
4. The time-cap line, alone.
5. Otherwise, per checked property without evidence, in the review's order:
   - one line per wrong implementation whose scenario is a scenario of it, in file order;
   - or, when there is none, one line saying so.

Every value is whitespace-normalized. `<slug>` is the Intent's slug, `<i>` counts wrong implementations from 1, `<impl>` is `the correct implementation` or `wrong implementation "<name>"`, and `<step>` and `<failing step>` are step lines as acceptance.feature writes them, without indentation.
- `- controls.json: .kogen/intents/<slug>/controls.json does not exist; write it as {"correct":{"files":{"<path>":"<content>"}},"wrong":[{"name":"<name>","scenario":"<scenario>","step":"<step line>","files":{"<path>":"<content>"}}]}`
- `- controls.json: it is not valid JSON`
- `- controls.json: it must be an object with exactly the fields "correct" and "wrong"`
- `- controls.json: "correct" must be an object with exactly the field "files"`
- `- controls.json: wrong implementation <i> must be an object with exactly the fields "name", "scenario", "step" and "files"`
- `- controls.json: wrong implementation <i> needs a name that is not blank and differs from the others`
- `- controls.json: <impl>: "files" must map at least one path to its content`
- `- controls.json: <impl>: <path>: a replacement path must be relative and normal`
- `- controls.json: <impl>: <path>: a replacement may not write acceptance machinery (the .kogen/ and steps directories, the Acceptance: files, protected files, .git, node_modules or a Cucumber configuration file)`
- `- controls.json: <impl>: it has more than 8 files or 65536 bytes`
- `- controls.json: wrong implementation "<name>": scenario "<scenario>" is not a scenario of acceptance.feature`
- `- controls.json: wrong implementation "<name>": step "<step>" is not a step of scenario "<scenario>"`
- `- run cap: this candidate needs <n> runs (the correct implementation and <w> wrong ones), above the cap of 8 runs per candidate; nothing was run. Supply at most 7 wrong implementations.`
- `- time cap: the runs exceeded the total time cap of <T> seconds per candidate after <k> of <n> runs completed; the check stopped. Make the implementations or the steps faster.` (`<T>` is the cap in whole seconds.)
- Property lines are `- <property> (<kind>) | request: <request> | problem: <reason>`, as `propertyItemLine` writes them but without a confirmation. The reasons:
  - `no pass is reachable: with the correct implementation in place, scenario "<S>" fails at "<failing step>": <first line of its failure message>`
  - `no pass is reachable: with the correct implementation in place, the run did not report every scenario as passed or failed`
  - `no wrong implementation in controls.json names a scenario of it`
  - `wrong implementation "<name>" passes scenario "<S>"`
  - `wrong implementation "<name>" fails scenario "<S>" at "<failing step>", not at "<step>"`
  - `wrong implementation "<name>" fails scenario "<S>" without an assertion: <type>: <first line of its failure message>`
  - `wrong implementation "<name>" fails scenario "<S>" with an assertion thrown outside the Intent's step files, at <file>:<line>:<column>`
  - `wrong implementation "<name>" also fails scenario "<S2>", which exercises another property and not this one`
  - `wrong implementation "<name>": its run did not report every scenario as passed or failed`

  For a wrong implementation, the first rule of 1 to 6 that fails picks its reason, in the order of these reasons. Rule 6 names the first such scenario in the review's order.

The refusal text is this headline followed by the item lines, one per line, with `<N>` the number of item lines and `<T>` the time cap in whole seconds:

```
Wrong-implementation check: <N> problem(s) with whether acceptance rejects a wrong implementation of each property the request asks for; nothing was published. For each property the request or the answers ask for that a step can observe, .kogen/intents/<slug>/controls.json supplies a correct implementation, with which every scenario of the property must pass, and wrong implementations, each naming a scenario and the step of it that must reject it. At least one wrong implementation of each property must fail that step with an AssertionError thrown by a step file the Intent lists under Acceptance, after the earlier steps of that scenario passed, while every scenario of the other properties still passes. Kogen runs the candidate's acceptance once per implementation in an isolated copy, at most 8 runs and <T> seconds per candidate. Make the steps assert the property, or supply a plausible wrong implementation that only gets the property wrong; resubmitting this candidate unchanged is refused again.
```

How the refusal is wired: the lines join `contractConfirmed` as a fourth part after the property lines.
- `all()` lists them last.
- `contractFeedback` appends the headline and the lines after the existing parts.
- The refused candidate goes through the existing contract-refusal path: `sh.ready = nil`, `contractRefused[hash]`, `contractLines`, and `submission_refused` with gate `contract_review`.
- The not-ready lines and `contract-contradictions.json` (which `kogen intent approve` reads) contain the lines.
- A clean check leaves the branch to go on as today (observations to acknowledge, or `accepted: the Intent is ready`).

Record and progress:
- Before each run, Shape writes the progress line `kogen: wrong-implementation check: run <i> of <n> (correct implementation)` or `kogen: wrong-implementation check: run <i> of <n> (wrong implementation "<name>")`.
- For each check that had checked properties, it records one event `{"event":"wrong_implementation_check","package":"<hash>","result":"<clean|refused>","runs":<runs started>,"seconds":<wall seconds of the runs>,"problems":["<item line>"]}`.

Build notes:
- The acceptance harness compiles `shape/k_wrong_implementation_acceptance_test.go` from the Go source in `features/steps/k-shape-wrong-implementation__harness.js` through a Go overlay. Do not create that path, and do not use the `kwi` or `KWrongImplementation` identifier prefixes in the package.
- The harness sets `ShapeInput.WrongImplementationTimeCap` by name through reflect, so that it compiles on the unchanged tree. Keep that exact field name and type.
- Keep these shaping test helpers, which the harness calls, working with their current signatures:
  - the functions `newShpFx`, `shpSubmit`, `shpAnswer`, `shpCall`, `shpOutputs`, `contractKind` and `refSubmit`, the constant `shpSlug`, and the type `shpStep`;
  - the `shpFx` methods `write`, `exists`, `git` and `newSession`, and its fields `t`, `Root`, `Aux`, `S`, `Stderr`, `Client` and `Prov`;
  - the `shpProvider` fields `seq` and `route`, and its methods `requests` and `contractRequests`;
  - `Shape`, `ShapeInput`, `Refine`, `RefineInput`, `SubmitContractReviewTool` and `SubmitContractConfirmationTool`.
- A natural home is a new `shape/controls.go` (with the embedded preload `shape/controls_preload.js`), called from `checkSubmission`, with its tests in `shape/controls_test.go`.

## Decisions
- Before an Intent whose request asks for automatically observable properties publishes, its acceptance must reject at least one plausible wrong implementation of each such property at an assertion about that property, and a correct implementation must pass the acceptance for those properties. (source: request)
- The wrong and correct implementations are small, touch only implementation files, and never alter the user's project, the acceptance or the Intent. (source: request)
- A candidate whose wrong implementation passes, or whose acceptance has no reachable pass, is not published, and the refusal names the property and what to repair. (source: request)
- Running out of the run cap or the time cap refuses the candidate, and the refusal says which cap ran out. (source: request)
- Plain input/output requests and properties recorded as needing human evaluation trigger no runs and no model request. (source: request)

## Source comparison
- Addressed, Need 1: rules 1 to 6 decide whether a run of the candidate's own acceptance, with a wrong implementation in place, fails at an assertion about the property. Rules 4 and 6 exclude load errors, missing files, timeouts and unconditional throws. Scenarios "bound", "kept" and "evidence".
- Addressed, Need 2: the correct run must pass every scenario of the checked properties, or the candidate is refused with "no pass is reachable". Scenario "unreachable".
- Addressed, Need 3: Shape decides that the shaper supplies both implementations in controls.json. Size, path and isolation are bounded: at most 8 files and 65536 bytes per implementation, no acceptance-machinery path, runs only in a fresh copy outside the checkout.
- Addressed, Need 4: the refusal names each property without evidence, the wrong implementation and scenario involved, and the repair, through the contract check's refusal path. Scenarios "bound", "kept", "unreachable" and "evidence".
- Addressed, Need 5: at most 8 runs and 300 seconds of runs per candidate. Each cap refuses with its own named line, never silently. Scenario "caps".
- Addressed: only checked properties are judged, which means properties the review takes from the Request and the answers (`propertyCounts`). No wrong implementation is demanded for behaviour they do not state.
- Constraints addressed:
  - No runs and no model request for plain requests and human-evaluation properties. Scenario "plain": 2 shaper requests, 1 contract request and 2 acceptance runs, the counts observed on the unchanged tree.
  - Existing checks are unchanged, and the check only adds to them. Scenario "existing" runs without controls.json and passes on the unchanged tree. In it, the property check still refuses a candidate with no scenario for sub, naming "sub keeps working" with "no scenario exercises it", after 2 acceptance runs. The unpassable-step check still refuses "Then is-even: mul is fixed at 7" with "no pass is reachable", after 4 acceptance runs. Neither shows the new headline.
  - Shape only: Refine's `contractCheck` call runs no check. Scenario "existing" publishes a calc Intent through Shape with a review that reports no property, then refines it with a review that reports add, sub and mul. The refinement is ready and written, runs acceptance once, and its progress has no wrong-implementation check.
  - Runs use the configured setup and acceptance command and the candidate's step files in an isolated copy. The acceptance-run counts in scenarios "bound", "kept", "unreachable" and "caps" observe the configured command running once per implementation.
  - Evidence is bound to the package hash. Scenario "kept": a strict candidate passes the check, then a loosened candidate in the same run is refused with fresh runs.
- Synthetic acceptance cases 1 to 5 are scenarios "bound", "kept", "unreachable", "caps" and "plain"; scenario "existing" covers the constraints on existing checks and Refine. Each uses small synthetic projects, a scripted shaper and a scripted contract review, with a real Cucumber run and no live providers. Each observes publication, the refusal named in the result of the submission it is about, the not-ready lines, model request counts and acceptance-run counts. Case 3's step `f of 1000 returns both 2000 and 2001` fails for every implementation but uses the implementation, so the unpassable-step check's stand-ins do not refuse it first.
- Earlier attempt (Build 4c326cb3, blocked because acceptance could never pass): every step names the submission whose result it reads. A refusal is observed in the result of the submission that was refused. "publishes" is observed on Shape's final outcome and on the published Intent files, with no headline assertion on a later submission. The candidate that is first checked clean and then changed is scenario "kept": its first result has no refusal and its "change" result has the refusal. Before submitting, I ran the fixtures with the correct and each wrong implementation, under a freezing preload, and saw the intended outcome for every candidate (Feasibility).
- Previous attempt (Build 0a200932, review_blocked), finding 1, where any detail with AssertionError counted: rules 3 to 6 replace that. Scenario "evidence" covers both counterexamples:
  - a wrong implementation that throws its own AssertionError is refused "with an assertion thrown outside the Intent's step files";
  - a wrong implementation that fails an earlier unrelated step of the scenario is refused "at ..., not at ...".
- Previous attempt, finding 2, where a replacement body could disable assertions: the freezing preload and the acceptance-machinery path rule. Scenario "unreachable" has a correct implementation that overwrites `strictEqual` and `ok`, which is still refused with no reachable pass. Scenario "evidence" has a wrong implementation that sets `notStrictEqual` to `fail`, which is still refused as passing. Both run for real.
- Decided (Shape): controls.json lives in the Intent directory so the evidence is bound to the package hash. The shaper's prompt is not changed; the missing-file refusal line gives the exact form.
- Known limits, not claimed exhaustive:
  - Kogen cannot judge whether a supplied wrong implementation is semantically plausible beyond rules 1 to 6. The controls are part of the Intent that a person reviews at approval.
  - The preload freezes Node's assert modules only. Other in-process tampering, other assertion libraries, and repository scripts that the configured commands run are not guarded. A project protects such scripts with `protect` patterns, and replacements may not write protected paths.
  - A property the review omits, or an incomplete review, gets no runs, which is fail-open like the contract check.
- Out of scope (request): Refine.

## Tests
- A bound that a slowed wrong implementation passes is refused naming the bound, and a step measuring f against a same-run baseline publishes: the "bound" scenario (scenario)
- Kept A and B and new C: a step for A that passes whatever calc returns is refused naming only A, acceptance failing on each removal publishes after 4 runs, and a changed candidate gets fresh runs: the "kept" scenario (scenario)
- No reachable pass, also when the correct implementation overwrites assertions: the "unreachable" scenario plus a unit test of the incomplete-correct-run reason in shape/controls_test.go (scenario)
- The run cap refuses before any run, and the time cap stops the runs and refuses naming it: the "caps" scenario plus unit tests that the time cap wins over later problems and that no run starts after it (scenario)
- A plain request runs nothing more and makes the same model requests: the "plain" scenario (scenario)
- Existing checks keep working unchanged (the property check and the unpassable-step check refuse as before, with no new runs or headline): the "existing" scenario (scenario)
- Refine is unchanged: a refinement whose review reports properties is ready and written after one acceptance run, without the check: the "existing" scenario (scenario)
- An AssertionError thrown by implementation code, and a failure at an earlier unrelated step, are not evidence, and an implementation cannot disable or fake assertions: the "evidence" scenario plus real-run regression tests in shape/controls_test.go reproducing both findings of Build 0a200932 (regression)
- The evidence rules on known reports: a non-assertion failure (Error, a Cucumber step timeout, a missing module), an incomplete report, rule 6 isolation, outline steps, and stack frames under a symlinked temporary directory: unit tests in shape/controls_test.go (unit)
- controls.json parsing and validation, each controls.json line, and the acceptance-machinery path rule: unit tests in shape/controls_test.go (unit)
- Shaping accepts controls.json, and other paths get the new allowed-paths message: update the expectations in shape/shape_test.go and shape/propose_files_test.go (unit)
- Runs never write the checkout: an integration test with real runs comparing the checkout's files, index and HEAD before and after a check (integration)
- The refusal joins the contract refusal (cached for an unchanged package, in the not-ready lines and contract-contradictions.json), the wrong_implementation_check event and progress lines: unit tests in shape/controls_test.go (unit)
- These stay unchanged, and the existing shape tests pass unchanged except the allowed-paths message: readiness, the step self-check, the unwritten-tests, runner-observer, preparation and unpassable-step refusals, the contract check's findings and the property check, and the goldens (existing)

## Feasibility
- These are session probes, my own scratch runs and not Kogen probe records. They used an exported copy of HEAD 620be64 outside the checkout, with Node v24.21.0, go1.27.2 and Kogen's Cucumber 13.3.0 at $KOGEN_CUCUMBER.
- Cucumber's report: a failed step's `testStepResult.exception` has `type` "AssertionError" and a `stackTrace`. Its first `at` frame is the implementation file when implementation code throws (`calc.js:3:28`) and the step file when the step asserts (`features/steps/is-even__steps.js:4:10`). The paths are absolute under /private/tmp.
- Tampering:
  - Without a preload, a replaced module that assigns `require('node:assert/strict').strictEqual = () => {}` turned a failing scenario green.
  - With `NODE_OPTIONS=--require <preload>` freezing both assert modules and their functions, the assignment was ignored in sloppy-mode code, the scenario failed with an AssertionError from the step file, and Cucumber ran normally.
- Fixture check (scratch Go test, discarded): it ran every synthetic candidate with Cucumber, with the correct and each wrong implementation written over the fixture, under the freezing preload.
  - Every correct run passed all scenarios, except "fails for every implementation" and "correct implementation disables assertions". Both failed at "And is-even: f of 1000 returns both 2000 and 2001".
  - The slowed f failed the measured step with "f(1000) took 300 ms, above the baseline 0 ms plus 100 ms" and passed the returns-only step.
  - Each removal failed only its own scenario under strict steps. "add removed" passed the loose add step.
  - The throwing add failed with an AssertionError at frame calc.js. The arity change failed at "Then is-even: calc takes 3 arguments". The tampering add passed.
  - Without the preload, the tampering add failed the loose step with an AssertionError framed in the step file, and the disabling correct implementation passed. So a check without the freeze would publish both, and these journeys tell it apart.
  - A first version of the loose step read `assert.notStrictEqual` before `require` patched it. That made the tampering journey unable to tell the two apart, and I fixed it to call the module first.
- Unchanged tree, the submitted feature with the project's acceptance command:
  - 7 scenarios: 5 failed at their first refusal assertion, because Shape rejects the controls.json path ("shaping may write only ..."). The kept "plain" and "existing" scenarios passed.
  - 56 steps: 28 passed, 5 failed, 23 skipped, in about 12 s with a warm build cache.
  - In "existing": the property-check journey ran acceptance 2 times. The unpassable journey ran it 4 times, since readiness runs for each of its 3 submissions. The refinement ran it once and ended ready with the refined intent.md written.
  - The plain journey published with 2 shaper requests, 1 contract request and 2 acceptance runs (the runner check and readiness).
  - A journey refused at validation ran acceptance once (the runner check).
- The approval attempt bounds one acceptance command at 120 s (verify.StepTimeout). The journeys of a case therefore run in parallel subtests, and the time-cap journey uses a 2-second cap.
- evidence: {"id":"wrong-implementation-observer","kind":"observer","inputs":["features/steps/k-shape-wrong-implementation__steps.js","features/steps/k-shape-wrong-implementation__harness.js"],"observers":["features/steps/k-shape-wrong-implementation__steps.js"],"checks":[{"name":"good","command":"node features/steps/k-shape-wrong-implementation__steps.js good","exit":0,"output":"accepted\n"},{"name":"bad","command":"node features/steps/k-shape-wrong-implementation__steps.js bad","exit":1,"output":"rejected\n"}]}
