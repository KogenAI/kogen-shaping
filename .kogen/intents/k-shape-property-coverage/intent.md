# Check acceptance covers requested properties

Slug: k-shape-property-coverage
Model: not applicable — one stateless judgement of a candidate inside the existing contract check; its refusals reuse that check's refusal record and the bounded correction loop, with no new workflow state
Acceptance:
- .kogen/intents/k-shape-property-coverage/acceptance.feature
- features/steps/k-shape-property-coverage__steps.js
- features/steps/k-shape-property-coverage__harness.js

K-SHAPE-PROPERTY-COVERAGE in kogen-shaping, the first of two items, before K-SHAPE-WRONG-IMPLEMENTATION. Sometimes a request asks for a property of the result that is not a plain input/output pair: a quality, a bound, a comparison with the old behaviour, or a behaviour that must be kept. Today `kogen intent shape` can then publish acceptance that does not test that property, or tests it in a way that any change passes. Observed forms: a kept behaviour has no scenario; a property step is red on the unchanged tree only because a file, module or command it loads does not exist yet; a step only checks that builder-written tests exist or pass in some number; or a property no automatic step can observe is replaced by a figure, log line or document sentence that a scenario claims proves it. The testing rules state the rule in prose ("acceptance that exercises the central claim", "observable outcomes"), but nothing checks it. The contract check (shape/contract.go) looks only for expectations no correct implementation can pass, and readiness only confirms that steps run and are red on the unchanged tree.

After this change, before Shape publishes an Intent, its existing contract review also identifies every property the request and the caller's answers ask for. Kogen then refuses a candidate whose acceptance does not cover them. The refusal goes back to the shaper as repair feedback, like the contract check's other refusals: nothing is published, an unchanged resubmission is refused again, and when the correction allowance runs out Shape ends with `intent/not_ready` "the Intent is not ready to build". This adds to the existing checks and replaces none. Readiness, the unwritten-tests, runner-observer and unpassable-step refusals, and the contract check's contradiction and environment findings keep working unchanged. Only Shape changes. Refine (a separate item gives it these checks), `kogen intent approve`, the step self-check and kogen-verification are unchanged. No acceptance runs beyond what readiness already runs; executing wrong implementations is the next item. The Shaper's prompt, policy and tool definitions do not change, so shape/testdata/shaper_policy_golden.json and the force goldens stay byte-identical. Do not regenerate them.

Where it runs, and its cost: inside `(*shaping).contractCheck`. That check already runs once for each newly checked candidate package, after readiness accounted for it (`sh.lastFresh && !sh.contractChecked[hash]` in `checkSubmission`). It adds no model request. Properties are identified by the existing review request (`submit_contract_review`). Property problems are verified by the existing confirmation request (`submit_contract_confirmation`), which is made only when there is something to confirm. A plain input/output request asks for no property, so its review answers `"properties": []` and nothing is confirmed. It keeps today's route, outcome and number of model requests.

Review input: the review's user input (`contractReviewInput`) gets two more sections after `Readiness observations`. They come in this order and use the same `# <name>` form, with the content's trailing newlines removed. The sections before them stay byte-identical, and the confirmation input stays the review input followed by `# Findings to verify`.
- `# Request`: the original request of this shaping (`sh.request`; on a run that answers questions it is still the original request).
- `# Questions and answers`: the earlier questions with the caller's answers, exactly as the shaper reads them (`roundsText(st.Rounds)`, kept as `sh.earlier`), or `none` when there are none. For example:

```
# Questions and answers

Round 1:
1. No automatic step can tell whether a greeting feels welcoming. How is that judged?
   a) Human judgement is fine: a person evaluates it
   b) Define a measurable rule instead
   answer: 1: a
```

Review instructions: `contractReviewInstructions` keeps its lines, except that the empty answer in its last line becomes `{"findings":[],"environment":[],"properties":[]}`. These three lines are added at the end:
- `Separately, identify the properties the Request and the Questions and answers ask for: a quality, a bound, a comparison with the old behaviour, or a behaviour that must be kept. Take them from the Request and the answers themselves, not from what the candidate declares. A plain input/output request (a command, call or input with its expected output) asks for none: answer properties [].`
- `For each property give property (a short name), request (the exact text copied from the Request or the Questions and answers that asks for it), kind (kept for a behaviour that must be kept, new otherwise), observable (false only when no automatic step can observe it, such as a subjective quality), scenarios (the names, exactly as in acceptance.feature, of the scenarios that exercise it; for a property that is not observable, the scenarios that claim it holds, also through a figure, log line or document sentence standing in for it), decision (for a property that is not observable, the exact text of the candidate intent.md Decisions line that records it as needing human evaluation; otherwise empty) and issues.`
- `An issue is a scenario of the property whose evidence does not establish it, with problem, scenario, quote and why. Its problem is missing when a new property's scenario is red on the unchanged tree only because a file, module, function, command or test it needs does not exist yet (quote: that failure, copied exactly from the readiness observations), or test_count when the step's only evidence is that tests the Build writes exist or pass in some number and the step does not itself assert the property (quote: that code, copied exactly from the step file). A step that first gives the missing piece a do-nothing form and then fails on the property's behaviour has no issue.`

Confirmation instructions: `contractConfirmationInstructions` gets one line at the end:
- `For a property problem, answer confirmed only when the Request or the Questions and answers ask for that property and the candidate has the stated problem; otherwise answer refuted.`

Schema: the `submit_contract_review` schema stays strict, with `additionalProperties` false everywhere. It gets a third required field, `properties`: an array of objects with exactly these required fields:
- `property` (string), `request` (string) and `decision` (string)
- `kind`: `"new"` or `"kept"`
- `observable` (boolean)
- `scenarios` (array of strings)
- `issues`: an array of objects with exactly the required string fields `problem` (`"missing"` or `"test_count"`), `scenario`, `quote` and `why`

Parsing (`parseContractReview`):
- A payload with exactly `findings` and `environment` is valid and has no properties. The scripted fake's default answer `{"findings":[],"environment":[]}` and the existing scripted answers keep working.
- A payload with exactly `findings`, `environment` and `properties` is valid when every property and issue has exactly the fields above, with the right types and values.
- Anything else is invalid. For unexpected top-level fields the reason is `the payload must have exactly the fields "findings" and "environment", and optionally "properties"`.
- An invalid payload leaves the check incomplete exactly as today and refuses nothing. The progress line is `kogen: contract check incomplete (invalid submit_contract_review payload: <reason>); the candidate goes on`.

What Kogen decides: Kogen alone decides which reported properties count and what is wrong with them. Text comparisons normalize whitespace (`normalizeWhitespace`, `containsNormalized`).
- A property counts only when `property` and `request` are not blank and `request` is contained in the Request or in the Questions and answers text. Text that only the candidate's own files contain does not count. A property that does not count refuses nothing.
- A scenario is declared when its name is a scenario or outline name of the candidate acceptance.feature (`ev.scenarios`). It failed when the readiness result has a failed scenario of that name (`ev.failures`).
- An observable property that counts has these problems, in this order, where D is its listed scenarios that are declared (in listed order, without repeats):
  1. `uncovered`: D is empty.
  2. `passes_unchanged`: its kind is new, D is not empty and no scenario of D failed.
  3. `missing`, once per issue with that problem, when all of these hold: the kind is new; the issue's scenario is in D and failed; its quote is not blank and is contained in a failure detail of that scenario.
  4. `test_count`, once per issue with that problem, when both of these hold: the issue's scenario is in D; its quote is not blank and is contained in a step file the Intent lists under Acceptance (not acceptance.feature).
  Any other issue is ignored.
- A property that counts and is not observable has these problems (its issues are ignored):
  1. `claimed`, once per listed scenario that is declared (in listed order, without repeats).
  2. `unrecorded`: `decision` is blank, or no line of intent.md's `## Decisions` section both contains `decision` and contains `needs human evaluation` in any letter case. The section is the lines after `## Decisions` up to the next line that starts with `## `.
- Problems are ordered by property, in the review's order, and within a property as listed above. Their reasons, with `<S>` the scenario and `<Q>` the quote (whitespace normalized):
  - `uncovered`: `no scenario exercises it`
  - `passes_unchanged`: `every scenario that exercises it passes on the unchanged tree, so none is red at an assertion about it`
  - `missing`: `scenario "<S>" is red on the unchanged tree only because something it needs is missing: <Q>`
  - `test_count`: `scenario "<S>" only checks that tests the Build writes exist or pass in some number: <Q>`
  - `claimed`: `no automatic step can observe it, but scenario "<S>" claims it holds`
  - `unrecorded`: `no automatic step can observe it, and no Decision records it as needing human evaluation`

Confirmation: the confirmation call is made when there is at least one evidenced contradiction, evidenced environment finding or property problem. Property problems are numbered after the evidenced contradictions and environment findings, each as the line `<n>. property: <property> (<kind>) | request: <request> | problem: <reason>`. Verdicts count as today:
- The first verdict for each number counts.
- A problem without a counted verdict is refuted.
- An invalid confirmation payload leaves the check incomplete and refuses nothing.

Only confirmed problems refuse. The item line of one is `- <property> (<kind>) | request: <request> | problem: <reason> | confirmed: <why>`, with values whitespace-normalized.

Refusal: a candidate with any confirmed contradiction, environment finding or property problem is refused through the existing path.
- `contractConfirmed` gains the property item lines. `all()` lists contradictions, then environment findings, then property problems.
- `submission_refused` records them with gate `contract_review`.
- `contractRefused` returns the same text for the unchanged package.
- The not-ready lines and `contract-contradictions.json` (which `kogen intent approve` reads) contain them.

The refusal text is today's contradiction part and environment part. When there are confirmed property problems, it then has this headline, followed by their item lines, one per line:

```
Property check: <N> problem(s) with how acceptance covers the properties the request asks for; nothing was published. Every property the request or the answers ask for that a step can observe needs a scenario that exercises it, and a new one's scenario must be red on the unchanged tree at an assertion about that property, not only because a file, module, function, command or test is missing: give the missing piece a do-nothing form first, then fail on its behaviour. A step whose only evidence is that tests the Build writes exist or pass in some number is not evidence for a property unless it asserts the property itself. A property no automatic step can observe needs a Decisions line saying it needs human evaluation, with its source, and no scenario may claim it holds, also not through a figure, log line or document sentence. Resubmitting this candidate unchanged is refused again.
```

Record: the `contract_review` event keeps its fields and meaning. Its `findings`, `evidenced` and `confirmed` still count only contradictions and environment findings. Its `result` is `refused` whenever anything is confirmed, a property problem included. When the review payload has at least one property, Shape records one more event after the check:
`{"event":"property_check","package":"<hash>","properties":[{"property":"<p>","request":"<r>","kind":"<kind>","observable":<bool>,"scenarios":["<s>"],"evidenced":<bool>}],"problems":[{"property":"<p>","reason":"<uncovered|passes_unchanged|missing|test_count|claimed|unrecorded>","scenario":"<s or empty>","verdict":"<confirmed|refuted>","confirmation":"<why>"}]}`
`verdict` and `confirmation` are null when the confirmation was incomplete. `confirmation` is also null for a problem refuted because it had no counted verdict. No Intent section or file format is added: a property that needs human evaluation is recorded as an ordinary Decisions line.

Build notes:
- The acceptance harness compiles `shape/k_property_coverage_acceptance_test.go` from the Go source in `features/steps/k-shape-property-coverage__harness.js` through a Go overlay. Do not create that path, and do not use the `kpc` or `KPropertyCoverage` identifier prefixes in the package.
- Keep these shaping test helpers, which the harness calls, working with their current signatures:
  - the functions `newShpFx`, `shpSubmit`, `shpAnswer`, `shpCall` and `shpOutputs`, and the type `shpStep`
  - the `shpFx` methods `write`, `exists`, `git` and `shape`, and its fields `Aux` and `Stderr`
  - the `shpProvider` fields `seq` and `contract`, and its methods `requests` and `contractRequests`
  - the constants `SubmitContractReviewTool` and `SubmitContractConfirmationTool`
- A natural home is a new `shape/property.go`, called from `contractCheck`, with its tests in `shape/property_test.go`.

## Decisions
- Kogen identifies the properties with the existing contract review request and verifies their problems with the existing confirmation request, so plain work makes no extra model request. (source: request)
- A refused candidate is not published. The refusal names each property and why, and returns to the shaper as repair feedback through the existing contract-check refusal. (source: request)
- A property no automatic step can observe is recorded as a Decisions line containing "needs human evaluation" with its source tag, and no scenario may claim it holds. (source: request)
- A step that first gives the missing piece a do-nothing form and then fails on its behaviour stays acceptable. (source: request)

## Source comparison
- Addressed, Need 1 (properties come from the request and the answers themselves, kept behaviours included; a plain input/output request has none): the Request and Questions and answers review sections, the added instruction lines, and the rule that a property counts only when its request text is in them. Case 1 includes a property that only the candidate states ("mul is memoized"); it refuses nothing. Case 4 checks that the review reads the answers, and case 5 the plain request.
- Addressed, Need 2 (each observable property is exercised by a scenario, or the refusal names it): `uncovered`; scenario 1, where the candidate testing only C is refused naming "add keeps working" and "sub keeps working" and the candidate testing A, B and C publishes.
- Addressed, Need 3 (a new property's scenario is red at an assertion about it; red only because something is missing is refused with that reason; a do-nothing form is fine): `missing` cites the readiness failure in the refusal. Scenario 3: a missing target file is refused with "Cannot find module '../../slug'"; the same step on a do-nothing slug.js fails on its behaviour and publishes.
- Interpretation of Need 3: a new property whose declared scenarios all pass on the unchanged tree is not red at an assertion about it either, so it is refused as `passes_unchanged`. This is not among the synthetic cases and is unit-tested.
- Addressed, Need 4 (a step whose only evidence is that builder-written tests exist or pass in some number is refused unless it asserts the property): `test_count`; scenario 2, where the count-only candidate is refused and the candidate measuring f against the bound in the same run publishes.
- Addressed, Need 5 (a property no step can observe is recorded as needing human evaluation under a Decision or an answer; no scenario or stand-in claims it): `claimed` and `unrecorded`. Scenario 4 runs after the caller answered that human judgement is fine. The recorded and unclaimed candidate publishes, the welcoming-score candidate is refused, and a candidate without the Decision is refused too.
- Interpretation: Kogen recognizes the recorded Decision mechanically by the phrase "needs human evaluation" in a Decisions line, so no new section or format is needed.
- Addressed: refusals go back as repair feedback and nothing publishes (the not-ready lines are checked at exhaustion).
- Constraints addressed: the plain route is unchanged (scenario 5: 2 shaper requests and 1 contract request, the count observed on the unchanged tree); the existing checks are unchanged; the model is used only through existing requests; Shape only; no new acceptance runs; the record is small; no new Intent section or format.
- Synthetic acceptance: each of the five cases is its own scenario. Each uses small synthetic projects, a scripted shaper and a scripted contract review, with real Cucumber readiness and no live providers. Each observes publication, the refusal text, the not-ready lines and the model request counts. Expected values live in acceptance.feature; the harness only collects observations.
- Known limit, not claimed exhaustive: identifying properties is a model judgement, so a property the review omits is not checked. An incomplete review or confirmation refuses nothing, which is fail-open as the contract check is today.
- Out of scope (request): Refine; executing wrong implementations (K-SHAPE-WRONG-IMPLEMENTATION).

## Tests
- Properties come from the request and the answers, never from the candidate's own files: the "kept" and "human" scenarios check the review sections, and "mul is memoized" refuses nothing; plus unit tests of the counting rule in shape/property_test.go (scenario)
- An observable property, kept or new, without a declared scenario is refused naming it: the "kept" scenario plus unit tests (scenario)
- A step whose only evidence is a count of builder-written tests is refused, and a same-run measurement against the bound publishes: the "bound" scenario plus unit tests of test_count quotes against step files (scenario)
- A new property's scenario that is red only because something is missing is refused with that failure, and the same step failing on a do-nothing form publishes: the "missing" scenario plus unit tests of missing quotes against failure details (scenario)
- A new property whose declared scenarios all pass on the unchanged tree is refused as passes_unchanged: a unit test in shape/property_test.go (unit)
- A property no step can observe publishes when recorded as needing human evaluation and unclaimed, and is refused when a scenario claims it or no Decision records it: the "human" scenario plus unit tests of the Decisions-line rule (scenario)
- A plain request keeps its route and its number of model requests: the "plain" scenario (scenario)
- A review payload without properties is valid and has none, and a malformed property or issue leaves the check incomplete and refuses nothing: unit tests in shape/property_test.go (unit)
- Unit tests cover: confirmation numbering after contradictions and environment findings; refuted and unverdicted problems never refuse; the order of the refusal text; and the property_check event (unit)
- These stay unchanged, and the existing shape tests pass unchanged: readiness; the unwritten-tests, runner-observer and unpassable-step refusals; the contract check's contradiction and environment findings; the goldens (existing)

## Feasibility
- Session prototype, my own scratch run and not a Kogen probe record. It used an extracted copy of HEAD 4ded2c4 outside the checkout, with Node v24.21.0, go1.27.2 and Kogen's Cucumber at $KOGEN_CUCUMBER, and ran the submitted feature and step files with the project's acceptance command. All 5 scenarios failed at their first property assertion and no harness step failed (39 steps: 5 passed, 5 failed, 29 skipped), in about 25 s including the one-time overlay compile. The candidates that must be refused were published. Every review answer carrying properties was dropped as "contract check incomplete (invalid submit_contract_review payload: the payload must have exactly the fields "findings" and "environment"); the candidate goes on".
- The masked later steps reached their intended behaviour on the unchanged tree, according to the harness's JSON observations for every candidate:
  - The published candidates made 2 shaper requests and 1 contract request; the green human candidate made 3 and 1, counting its question run.
  - The kept add and sub scenarios passed and mul failed.
  - The slow f failed with "... above 100 ms".
  - The absent slug failed with "Cannot find module '../../slug'", and the do-nothing slug with "slug returned ...".
  - `node --test` with no test files printed "ℹ pass 0", so the count-only step failed at its own assertion.
  - A candidate that readiness finds green ends Shape at once, so it gets no submission result.
- The overlay key is relative (`shape/k_property_coverage_acceptance_test.go`): an absolute key did not apply under macOS's /tmp to /private/tmp link.
- evidence: {"id":"property-coverage-observer","kind":"observer","inputs":["features/steps/k-shape-property-coverage__steps.js","features/steps/k-shape-property-coverage__harness.js"],"observers":["features/steps/k-shape-property-coverage__steps.js"],"checks":[{"name":"good","command":"node features/steps/k-shape-property-coverage__steps.js good","exit":0,"output":"accepted\n"},{"name":"bad","command":"node features/steps/k-shape-property-coverage__steps.js bad","exit":1,"output":"rejected\n"}]}
