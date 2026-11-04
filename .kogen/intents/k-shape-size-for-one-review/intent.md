# Refuse Intents too large for one review

Slug: k-shape-size-for-one-review
Model: not applicable — one deterministic, stateless check of a candidate package inside the existing submission loop; its refusals reuse the existing refusal record and bounded correction loop
Acceptance:
- .kogen/intents/k-shape-size-for-one-review/acceptance.feature
- features/steps/k-shape-size-for-one-review__steps.js
- features/steps/k-shape-size-for-one-review__harness.js

Today `kogen intent shape` can publish an Intent whose change is larger than one review can hold. Only the review stage finds out, after the Build was paid for: kogen-review measures the candidate's complete one-session review prompt against `tools.InputBudgetBytes` (kogen-loop's `provider.InputBudgetBytes`, 524288 bytes, kept as `reviewCeiling` in kogen-review's review/parts.go) and above it reviews the candidate in ordered parts ("the complete review prompt would be X bytes, over the 524288-byte ceiling"). Nothing at shaping time looks at the size of what the Intent asks for.

After this change Shape runs a deterministic scope check on every candidate package before it can be made ready. It estimates the bytes one review of the Intent needs from what the Intent itself states and compares the estimate with the review stage's own bound, read from that same named setting. Above the bound the candidate is refused through the existing refusal path. The refusal names the counted items, the estimate and the bound, and tells the shaper to split the work into an ordered staged plan or to narrow it to the requested behaviour. At or below the bound nothing changes: no feedback line, no progress line, no event and no model request. Only Shape changes.

Bound: `tools.InputBudgetBytes` as compiled in (import `kogen-loop/tools`; it is kogen-loop's `provider.InputBudgetBytes`). Do not copy its value into a shaping constant, and do not use a per-model or per-role budget. The bound is inclusive like the review stage's: a unit fits when its estimate is at most the bound.

What is counted, from the candidate package as written and formatted for the check, and from the checkout's HEAD:
- Intent package: intent.md, every path under `Acceptance:` in listed order (acceptance.feature and each harness or step file), and the model file when the Model line declares one. Each counts the byte length of its written content. Every review carries the whole package.
- Named files: a span is the text between a pair of single backticks on one line: a backtick, one or more characters that are neither a backtick nor a line break, and a backtick, matched left to right (Go regexp "\x60([^\x60\n]+)\x60"; a fence of three backticks matches nothing). A span is a named file when its text equals exactly the path of a blob in the HEAD tree (as `git ls-tree -r -l HEAD` lists it), is not under `.kogen/` and is not an Intent package path. It counts that blob's size at HEAD. A path named more than once counts once, in order of first appearance. A path that is not at HEAD (a file the Build will add) is not counted.
- Areas: the distinct directories of the counted named files (`path.Dir`; `.` for a file at the root).
- Scenarios: the scenarios `project.CheckIntent` returns for the candidate (an outline counts once per Examples row).
- Acceptance harnesses: the `Acceptance:` paths other than acceptance.feature (`intent.StepFiles()`).
- The estimate of a unit is the sum of the Intent package bytes and the bytes of the unit's named files.

Units:
- Without a `## Stages` plan (`project.ParseStages` returns nil) there is one unit, the Intent, with every named file.
- With a valid staged plan there is one unit per stage, in plan order, and the Intent as a whole is not checked. A stage's named files are the named files whose path is a span (same rule) of that stage's `goal`, plus every named file that no stage goal names: such a file counts in every stage. A stage's scenarios and harnesses are those of the whole package, because every stage carries the whole Intent package.

Refusal: when any unit's estimate exceeds the bound, the candidate is refused and nothing is published. The submission result is, for each unit over the bound in order, its unit line and then its item lines, followed by the guidance line, joined by "\n":
- Unit line: `Scope check: <unit> is estimated at <E> bytes of one review (scenarios <S>, acceptance harnesses <H>, named files <F> in areas <A>), over the <L>-byte bound of one review (tools.InputBudgetBytes).` Here <unit> is `the Intent` or `stage "<id>"`, <F> and <A> are counted for that unit, and <L> is the bound.
- Item lines: one per Intent package file in the order above, `- <path>: <n> bytes (Intent package)`, then one per named file of the unit in first-appearance order, `- <path>: <n> bytes (named file, area <dir>)`.
- Guidance line, exactly: `Above the bound, split the work into an ordered ## Stages plan of at least two stages, each with a real gate in its checks and each inside the bound, or narrow the Intent to the requested behaviour. A stage's goal names in backticks the files that stage changes; a named file that no stage goal names counts in every stage, and every stage carries the whole Intent package. Do not satisfy the bound by deleting requested behaviour: every behaviour and example the request asks for keeps its scenario. Resubmitting this candidate unchanged is refused again.`

For example, the first submission of the request's second synthetic case (40 scenarios over 12 areas, one step file) begins:

```
Scope check: the Intent is estimated at <E> bytes of one review (scenarios 40, acceptance harnesses 1, named files 12 in areas 12), over the 524288-byte bound of one review (tools.InputBudgetBytes).
- .kogen/intents/is-even/intent.md: <n> bytes (Intent package)
- .kogen/intents/is-even/acceptance.feature: <n> bytes (Intent package)
- features/steps/is-even__steps.js: <n> bytes (Intent package)
- alpha/module.js: <n> bytes (named file, area alpha)
```

The refusal uses the existing path: `submission_refused` with gate `scope` and the result's lines as its problems (the existing record bounds apply), an invalid outcome with those problems, the existing "Unchanged:" note for an identical package, and the existing `intent/not_ready` end when the correction allowance runs out.

Where it runs: in `checkSubmission`, after the package is written and formatted and after the existing `contractRefusal` lookup, before the evidence gate, the decision check, the step self-check and readiness. A refused candidate therefore costs no decision check, contract check or readiness run. No progress line is added.

What does not change: the Shaper's prompt, policy and tool definitions (shape/prompts; shape/testdata/shaper_policy_golden.json, prompts_golden.json, defs_golden.json and the force goldens stay byte-identical; do not regenerate them); the staged-plan format and its parsing (kogen-coordinator `project.ParseStages`); the existing gates and their order; Refine, `kogen intent approve`, the review stage and the Build (a staged Build's final whole review belongs to kogen-build and is unchanged).

Build notes:
- The acceptance harness compiles `shape/k_size_for_one_review_acceptance_test.go` from the Go source in features/steps/k-shape-size-for-one-review__harness.js through a Go overlay. Do not create that path, and do not use the `kss` or `KShapeSizeForOneReview` identifier prefixes in the package.
- To change the setting, the step file builds the same tests against a private copy of the pinned kogen-loop (from `go list -m -f '{{.Dir}}' kogen-loop`) whose line `const InputBudgetBytes = 524288` in provider/respond.go is replaced, through a private go.mod that is the checkout's go.mod with its `replace kogen-loop => ...` line pointing at the copy (Go refuses overlays of module-cache files). Keep that single replace line and read the bound from `tools.InputBudgetBytes` or `provider.InputBudgetBytes`, so a changed setting reaches the check.
- Keep these shaping test helpers, which the harness calls, working with their current signatures: `newShpFx`, `shpSubmit`, `shpAnswer`, `shpCall`, `shpOutputs`, `contractKind` and the type `shpStep`; the `shpFx` methods `write`, `exists`, `read`, `git` and `shape` and its fields `Aux`, `Prov`, `Stdout` and `Stderr`; the `shpProvider` field `route` and its methods `requests`, `decisionRequests` and `contractRequests`; the constants `SubmitContractReviewTool` and `SubmitContractConfirmationTool`.
- A natural home is a new shape/scope.go, with its unit tests in shape/scope_test.go.

## Decisions
- An Intent inside the bound keeps today's route: no new feedback line and no extra model request. (source: request)
- Above the bound the refusal names the counted items, with one line naming the counts and the bound, and tells the shaper to split the work into an ordered staged plan with a real gate per stage or to narrow the Intent to the requested behaviour, never by deleting requested behaviour. (source: request)
- A staged plan whose stages are each inside the bound is accepted, with nothing dropped. (source: request)
- The bound is the one named setting the review stage uses as its limit for one review, `tools.InputBudgetBytes` (524288 bytes, kogen-review's `reviewCeiling`); shaping invents no number of its own. (evidence: ran `go doc kogen-loop/provider InputBudgetBytes`; ran `grep -n "reviewCeiling = " /Users/kogen-runner/work.noindex/gomodcache/github.com/\!kogen\!a\!i/kogen-review@v0.0.0-20261103161351-e5005fa1daf8/review/parts.go`) (source: request)

## Source comparison
- Interpretation (Need): the review stage states its limit only in bytes, the inclusive one-session prompt ceiling `tools.InputBudgetBytes`; it states no count of scenarios, files or harnesses. So the estimate is in bytes, built from the items the request names: the scenarios and harnesses (the Intent package every review carries) and the files the Intent names (their HEAD size). The refusal names those counts. The estimate is a lower bound of the one-session review prompt, which carries the whole package and the full text of every changed file, plus diffs and check results the estimate omits. A refusal therefore shows a review that cannot be one piece, while an estimate inside the bound does not prove one.
- Addressed, synthetic case 1 (3 scenarios, 2 files: accepted, no new feedback line) and case 6 (no extra model request): scenario 1, same route and the request counts observed on the unchanged tree (2 shaper, 2 decision check, 1 contract check). This is kept behaviour and passes on the unchanged tree.
- Addressed, case 2 (40 scenarios in 12 unrelated areas refused with a line naming the counts and the bound; the same package as a three-stage plan accepted): scenario 2.
- Addressed, case 3 (narrowed to the requested behaviour: accepted): scenario 3.
- Addressed, case 4 (every behaviour needed: staged plan accepted, nothing dropped; deleting requested behaviour does not pass): scenario 4. The scope refusal forbids deleting requested behaviour, and a candidate that drops the requested example "lima behaviour 36" is refused by the existing property check (scripted review). The published plan keeps all 40 scenarios, that example included.
- Addressed, case 5 (bound from one named setting shared with the review stage; changing it changes the verdict): scenario 5 builds against a copy of kogen-loop with `tools.InputBudgetBytes` 1048576; the package refused at 524288 is published, and a heavier one is refused naming 1048576.
- Addressed: deterministic, no new model request, refusal before any Build or readiness run; a refused candidate makes no decision or contract check request (counts in scenarios 2 and 4).
- Interpretation: how a stage's share is known. The plan format is fixed (id, title, goal, checks), so a stage owns the files its goal names in backticks; unassigned files count in every stage, so a plan cannot hide scope. "A real gate per stage" is the existing nonempty-checks rule of the plan; the check cannot prove a gate is meaningful.
- Known limits, not claimed exhaustive: only files named in backticks and present at HEAD are counted, so an Intent that does not name its files is under-estimated. The request accepts a staged plan whose stages fit, but kogen-build still gives a staged Build one final whole review; that is unchanged here.
- Out of scope: Refine, the Shaper's prompt and goldens, kogen-review, kogen-build and kogen-loop.

## Tests
- An Intent inside the bound keeps its route, its submission result and its model requests: the "small" scenario (scenario)
- An Intent over the bound is refused with the unit line naming its counts and the bound, its item lines and the guidance line, and its staged plan of stages inside the bound is accepted: the "wide" scenario (scenario)
- An over-scoped Intent narrowed to the requested behaviour is accepted: the "narrow" scenario (scenario)
- A request that needs every behaviour is accepted as a staged plan with every scenario, and a candidate that drops a requested example is not published: the "every" scenario (scenario)
- The bound is tools.InputBudgetBytes and changing it changes the verdict: the setting scenario (scenario)
- The estimate rules: an estimate equal to the bound fits and one byte more is refused; a repeated name counts once; spans not at HEAD, under .kogen/, naming directories or naming Intent package paths are not counted; areas are distinct directories with . for the root; outline rows count as scenarios; a declared model file counts in the package: unit tests in shape/scope_test.go (unit)
- Stage units: a goal-named file counts only in its stage, an unassigned file counts in every stage, a plan with one stage over the bound is refused naming only that stage, and with a plan the whole Intent is not checked: unit tests in shape/scope_test.go (unit)
- A refused candidate is recorded as submission_refused with gate scope, an identical resubmission gets the Unchanged note, and the refusal comes before the decision check and readiness: a Shape fixture test in shape/scope_test.go (integration)
- The existing gates, the goldens and Refine are unchanged: the existing shape tests pass unchanged (existing)

## Feasibility
- My own scratch runs, not Kogen probe records, against HEAD f20e1bd with Node v24.21.0, go1.27.2 and Kogen's Cucumber at $KOGEN_CUCUMBER. Nothing in the checkout was written.
- The harness compiled through the overlay. On the unchanged tree the cases small, wide and every took 3 to 6 s each and published every over-bound package. Requests (shaper, decision check, contract check) were 2/2/1, 3/3/2 and 4/4/4. In every, candidate c2 was refused by the existing property check with "- lima behaviour 36 returns built 36 (new) | request: lima behaviour 36 returns built 36 | problem: no scenario exercises it | confirmed: synthetic confirmation". So scenario 1 passes as kept behaviour, and scenarios 2 to 5 fail at "submission "c1" is refused by the scope check". The changed-tree counts are derived: c1 skips its decision and contract check requests.
- A Go overlay replacing kogen-loop's provider/respond.go was refused: "Files beneath GOMODCACHE ... must not be replaced". A `-modfile` build against a copied kogen-loop with InputBudgetBytes 1048576 compiled, and the harness saw that bound; on the unchanged tree wide-unstaged and heavy published under it.
- A Cucumber dry run of the feature with the step files bound all 55 steps of the 5 scenarios.
- The observer controls below run the step file's judgments on known observations. Good is a refused-then-staged run as the implemented check would produce it. Bad is the unchanged-tree observation and single-field corruptions: a missing or prefixed scope line, wrong counts or bound, a split line, wrong file bytes, a missing or scope-refused later submission, unpublished, wrong bound, a missing plan, a dropped scenario.
- evidence: {"id":"size-for-one-review-observer","kind":"observer","inputs":["features/steps/k-shape-size-for-one-review__steps.js","features/steps/k-shape-size-for-one-review__harness.js"],"observers":["features/steps/k-shape-size-for-one-review__steps.js"],"checks":[{"name":"good","command":"node features/steps/k-shape-size-for-one-review__steps.js good","exit":0,"output":"accepted\n"},{"name":"bad","command":"node features/steps/k-shape-size-for-one-review__steps.js bad","exit":1,"output":"rejected\n"}]}
