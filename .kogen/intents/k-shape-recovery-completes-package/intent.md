# Pre-check incomplete packages during recovery

Slug: k-shape-recovery-completes-package
Model: .kogen/intents/k-shape-recovery-completes-package/model.qnt invariants=InterceptsFollowRefusals,BudgetUnchanged,PendingOnlyAfterCompleteness,IncompleteBudgetEndsBlocked,TurnsBounded seed=7 samples=5000 steps=16
Acceptance:
- .kogen/intents/k-shape-recovery-completes-package/acceptance.feature
- features/steps/k-shape-recovery-completes-package__steps.js
- features/steps/k-shape-recovery-completes-package__harness.js

Priority P1: the paired runs of the shaping repair lost valid Intents (4 vs 6 of 11). Shape's shared submission handling (`shape/shape.go`: `submit`, `submitOne`, `validate`, `acceptFiles`, the native `converse` loop, `converseExternal` and the stop rendering in `shaping.shape`) refuses an incomplete package in two ways. It refuses an envelope that is not exactly one of a non-empty questions list, a non-empty files list or a blocked report at gate `validation`. It refuses a files package whose declared files do not exist at gate `intent`. Today the envelope refusal names the rule but not what was received. A missing `acceptance.feature` hides every other missing declared file. A resubmission of the same incomplete package spends another attempt and gets the same generic text. At the budget, the run ends with `shape/invalid_output`. A recovery turn should end only when its refusal is resolved, and a shaper that cannot complete the package should end with an honest blocked report. After this change (native and external Shape; Refine unchanged):

1. **Declared-file gaps.** The declared-file gaps of the current state come from the accumulated proposals and the checkout. A file is absent when it is neither an accumulated proposal nor a regular file in the checkout. The gaps, in this order:
   - only `intent.md does not exist` when `.kogen/intents/<slug>/intent.md` is absent;
   - otherwise `acceptance.feature does not exist` when `.kogen/intents/<slug>/acceptance.feature` is absent;
   - then `acceptance: <p> does not exist` for each other distinct path `<p>` of the Intent's `Acceptance:` list that is absent, in list order;
   - then `model: <p> does not exist` when the `Model:` line names a model file `<p>` that is absent.

   These are the coordinator's own line forms. A files submission refused at gate `intent` because of the coordinator's parse error `acceptance.feature does not exist` now records and shows all declared-file gaps as its problem lines, not that line alone. Example: intent.md alone, declaring the feature and `features/steps/is-even__steps.js`, gives `acceptance.feature does not exist` and `acceptance: features/steps/is-even__steps.js does not exist`. Every other `intent` refusal keeps the coordinator's lines, which already name each missing Acceptance path and the model.
2. **Envelope shape.** The envelope refusal keeps its line `submit takes exactly one of a non-empty questions list, a non-empty files list, or a blocked report`. It gains a second problem line `received: <shape>`. For each of `questions`, `files` and `blocked` present in the arguments, in that order, the shape names `<key> (list of <n>)` for a JSON array of n entries. Any other value is `<key> (object)`, `<key> (text)`, `<key> (number)`, `<key> (boolean)` or `<key> (null)`. The parts are joined with `, `. When none of the three keys is present, the line is `received: none of questions, files or blocked`. Examples: `received: questions (list of 1), files (list of 3)` and `received: files (text)`. Other validation problems (paths, question format, blocked reports) are unchanged.
3. **Completeness refusal.** A counted refusal is a completeness refusal when it is one of these:
   - the envelope refusal (gate `validation`, first problem line the rule);
   - a refusal at gate `intent` while the declared-file gaps are not empty.
4. **Pre-check before the next submission.** After each completeness refusal at attempt `<n>`, the next submit call is pre-checked before it counts. That call consumes the pre-check, whatever the result. The gaps are:
   - for an envelope that breaks the rule: the rule line and its `received:` line;
   - for a well-formed files list (`validateFiles` finds no problem): the declared-file gaps after the call's files are merged into the accumulated proposals. The merged files are kept as propose_files keeps them (custody folder, cumulative manifest), whatever the pre-check finds;
   - for questions, a blocked report, or files with validation problems: none, so the call is the next counted submission exactly as today.

   With no gaps, the call is the next counted submission exactly as today. With gaps, the call is not a submission. It uses no attempt number: the next counted submission takes the next number. It records no `submission_refused`. It changes neither the invalid nor the no-tool count, nor the last submission or the recovery state. Shape records `{"event":"submission_precheck","owner":"shape","attempt":<n>,"gaps":[<gap lines>]}` in `.kogen/local/shaping/<slug>/events.ndjson`. The call's output is these lines joined by newlines:
   - first, when the gaps equal the problem lines of attempt `<n>`: `not submitted: this is the same refusal as attempt <n>; these items are unchanged, and this call was not counted as a submission attempt:`; otherwise: `not submitted: the package still has these gaps, and this call was not counted as a submission attempt:`;
   - then the gap lines;
   - last, for files: `The files of this call were retained with your earlier proposals. Add what is missing with propose_files; the next submission counts as an attempt.`; for an envelope: `No proposals from this call were retained; earlier proposals are unchanged. The next submission counts as an attempt.`

   Native Shape ignores a later submit call in the same response as today (`ignored: only one submit counts per response`). External Shape returns the text to the same session with the cumulative proposal manifest, as it does for other feedback, and does not count it toward the invalid budget.
5. **Same refusal again.** A completeness refusal can have the same gate and problem lines as the refusal of the immediately preceding counted submission (attempt `<m>`). Its feedback then gets the line `This is the same refusal as attempt <m>; these items are unchanged:` right after its first line `invalid submission:`. The problem lines, the closing sentence and the `Unchanged:` line of an identical package follow as today. The recorded problem lines do not change.
6. **Blocked report at the budget.** When the invalid-submission budget ends on a completeness refusal, Shape returns `*sys.Error` `shape`/`incomplete_package` with exit 1 and detail `blocked: the shaper could not complete the package within <limit> attempts`. Its lines are `unresolved gap (attempt <n>, gate <gate>):`, then that refusal's problem lines as recorded, then the earliest-unresolved block and the `question check refusals:` line where they apply today. This replaces `shape/invalid_output: the shaper's output was not a valid Intent after <limit> attempts` and `intent/not_ready` for that case only. Every other budget end is unchanged.

Unchanged: the invalid-submission budget (`submissionRetryLimit`, default 3, and `SubmissionRetryLimit`) and the no-tool limit (3). Nothing is added to the configuration, flags or limits. These also stay as they are:
- the recovery inference, its message, its `recovery_turn` event and the forced inference;
- the no-tool stop, still `shape/invalid_output`; its `last refusal` block shows the recorded lines, which may now include the `received:` line or more missing files;
- the `Unchanged:` line;
- every other gate and its text; the question check; the protected-path blocked report;
- readiness, acknowledgment and approval;
- Refine;
- the Shaper's instructions and tool definitions (policy goldens byte-identical) and the force goldens.

A complete first submission takes the ordinary path: no pre-check and no extra request. The shaper can still end with a permissible question or a protected-path blocked report. No new submission envelope is added.

The Build updates the existing tests that encode replaced behavior and keeps their purpose. Where a test is about the generic invalid budget rather than incompleteness, its invalid submission becomes a non-completeness refusal (for example a file outside the allowed paths), so it still ends with `shape/invalid_output`. These are known:
- `TestShapeThreeInvalidSubmissionsStopShapingAndWriteNothing`;
- `TestShapeSubmissionEnvelopeProblems` ("empty questions", "blocked beside files");
- the "validation" row of `TestShapeRecordsEachRefusedSubmissionWithItsGate`;
- `retryShape` with `TestSubmissionRetryDefault`, `TestSubmissionRetryOne`, `TestSubmissionRetryFive` and `TestSubmissionRetryEvidence`, plus the external `both` case in `shape/submission_retry_test.go`;
- `TestShapeForceInvalidBudget`;
- the `TestShapeForceNoToolBound` cases that script `both`;
- `TestShapeStopsAddEarliestRefusalOnlyWhenItDiffers`;
- the external case "an invalid submission is retried in the same session and three end shaping" in `shape/lineup_shape_test.go`;
- any other existing test whose script repeats a malformed envelope or an incomplete package into the budget.

## Acceptance harness
- `features/steps/k-shape-recovery-completes-package__steps.js` writes the Go source exported by `features/steps/k-shape-recovery-completes-package__harness.js` to a temporary file. It overlays that file as `shape/ksr_acceptance_test.go` and builds once per process with `go test -c -vet=off -overlay <tmp>/overlay.json -o <tmp>/shape.test ./shape`. Then it runs `<tmp>/shape.test -test.run ^TestKSRAcceptance$ -test.count=1 -test.v -test.timeout=150s` in `shape/` with `KSR_SCRIPT` set to the journey. Each run prints one `KSR <json>` line with these fields:
  - `exit` and `stop` (the rendered error);
  - `requests` (shaper requests, without contract, decision or question-check requests);
  - `recovery` (requests whose input gains a user message starting `Recovery before submitting again:`);
  - `outputs` (tool outputs by call id in the last shaper request);
  - `refused` and `prechecks` (the `submission_refused` and `submission_precheck` events without `event`);
  - `invalid` (`invalid_submissions` of `shaping_turn_counts`);
  - `intent_written`, `published` and `unpublished`.
- The Build must not create `shape/ksr_acceptance_test.go`, and must not use the `ksr` or `KSR` identifier prefixes or `TestKSRAcceptance` in the shape package. It keeps these test helpers with their names and behavior:
  - `responsesWire` (`calls`, `prose`, `newShape`) and `forceConversation`;
  - `shpCall`, `shpStep`, `shpQuestion`, `shpScopedIntentMD`, `shpFeature` and `shpSteps`;
  - `shpShaperRequests`, `shpOutputs` and `shpNamed`;
  - `shpFx` (`setReport`, `shape`, `events`, `exists`, `read`) and `shpProvider` (`seq`, `requests`).
- Journeys run in the synthetic is-even repository on the Responses wire with a green replayed report. Intent I is `shpScopedIntentMD`, declaring feature F and step file S `features/steps/is-even__steps.js`. `both` has one question and the files I, F and S.
  - `missing-step-written`: read README and submit I+F (refused, gate `intent`); propose_files S; submit S (accepted).
  - `missing-step-prose`: submit I+F (refused); prose; submit I+F again (pre-checked, not counted); propose_files S; submit S (accepted).
  - `intent-only`: submit I (refused, naming both missing files); submit F (pre-checked: S still missing); submit S (accepted).
  - `envelope-repaired`: submit `both` (refused, gate `validation`); submit I+F+S (accepted).
  - `envelope-repeated`: submit `both` (refused, attempt 1); prose; then `both` four more times. The first and third are pre-checked; the second and fourth are refused at attempts 2 and 3, and the budget ends with the blocked report.
  - `valid-first`: one complete submission (accepted).

## Decisions
- Refusal feedback names exactly what is missing or malformed: each declared file that does not exist, and the broken envelope rule with the received shape (source: request)
- Before the next submission, Shape checks the conditions it would refuse on (declared files exist, envelope shape valid). It returns the remaining gaps inside the recovery turn without spending an attempt (source: request)
- A refusal that recurs unchanged tells the shaper so and repeats the unchanged items, not just the generic message (source: request)
- Recovery stays within the existing attempt budget; no timeout, attempt or limit is added or raised (source: request)
- A shaper that cannot complete the package ends with a blocked report naming the unresolved gap, never with invalid output for a malformed envelope (source: request)
- A complete first submission is accepted with no recovery turn and no extra model request (source: request)
- Acceptance uses a scripted shaper in a small synthetic repository, without live model calls (source: request)

## Source comparison
- Addressed: the Problem, a shaper that resubmits the same incomplete package (missing acceptance or step file, or a malformed envelope) until the budget ends with invalid output. Points 4 to 6 cover it, with scenarios "After prose in recovery the unchanged package gets its remaining gap without spending an attempt" and "A repeated malformed envelope is reported unchanged and ends with a blocked report at the budget".
- Addressed: Need bullet 1 by points 1 and 2; scenarios "Every missing declared file is named and the pre-check returns the gap that remains" and "A malformed envelope names the rule and the received shape and the repaired files are accepted".
- Addressed: Need bullet 2 by point 4 (one pre-check per completeness refusal, checked on the next submit call); Need bullet 3 by point 4's unchanged first line and point 5; Need bullet 4 by point 6 and the unchanged budget and limits, with the model's bound of 11 prose answers, submit calls and pre-checks at the default limits.
- Addressed: Synthetic acceptance 1 by `missing-step-written`; 2 by `missing-step-prose`; 3 by `envelope-repaired`; 4 by `envelope-repeated`; 5 by `valid-first`. `intent-only` adds the missing-feature case of Need bullet 1.
- Conflicting literals: the request's `features/x.feature` and `features/steps/x.js` cannot form a package Shape accepts. The `Acceptance:` list must start with `.kogen/intents/<slug>/acceptance.feature`, and step files must be named `<steps_dir>/<slug>__<suffix>.js`. The journeys use the is-even fixture's `.kogen/intents/is-even/acceptance.feature` and `features/steps/is-even__steps.js` in their place.
- Decided: "explicitly reports it cannot be resolved" uses the existing outcomes (a permissible question, a protected-path blocked report), or ends at the budget with point 6's blocked report. A new self-reported envelope would weaken the protected-path rule of the `blocked` report, so none is added.
- Decided: the pre-check runs on the next submit call, the moment a doomed resubmission would spend an attempt. It runs once per counted completeness refusal, so it adds at most `limit - 1` model requests and stays bounded by the existing budget. Prose in a recovery turn keeps the existing recovery inference, whose message already quotes the refusal's lines.
- Kept: the no-tool stop stays `shape/invalid_output`. The request's blocked-report bullet concerns the attempt budget, and the no-tool stop already names the last refusal and the shaper's last answer.
- Superseded: the generic budget stop and the single-line refusals for incomplete packages, pinned by the existing tests listed above for update. Earlier Intents' packages (for example `.kogen/intents/shape-repair-before-terminal-submit/`) are history and are not edited.

## Tests
- Missing declared files are all named when the feature is missing: acceptance "Every missing declared file is named and the pre-check returns the gap that remains"; add a focused test with a missing model and two missing step files in Acceptance order (integration)
- The envelope refusal names the received shape: acceptance "A malformed envelope names the rule and the received shape and the repaired files are accepted"; update `TestShapeSubmissionEnvelopeProblems` and add cases for `received: files (text)` and `received: none of questions, files or blocked` (integration)
- The next submit after a completeness refusal is pre-checked, returns the gaps without spending an attempt and keeps the call's well-formed files: acceptance "After prose in recovery the unchanged package gets its remaining gap without spending an attempt" and "Every missing declared file is named and the pre-check returns the gap that remains"; add an external Shape test that the pre-check text reaches the same session and is not counted (integration)
- A pre-check that finds no gap, or a non-completeness refusal, leaves the next submission counted as today: acceptance "The refusal names the missing step file and the step written in recovery is accepted" and "A malformed envelope names the rule and the received shape and the repaired files are accepted"; update `TestShapeRecordsEachRefusedSubmissionWithItsGate` (integration)
- A recurring identical completeness refusal says so and repeats the unchanged items: acceptance "A repeated malformed envelope is reported unchanged and ends with a blocked report at the budget" (scenario)
- The budget ending on a completeness refusal is the blocked report, other budget ends stay `shape/invalid_output`: acceptance "A repeated malformed envelope is reported unchanged and ends with a blocked report at the budget"; update `TestShapeThreeInvalidSubmissionsStopShapingAndWriteNothing`, `retryShape` and the listed budget tests; add a test that a missing step file at the budget ends with `shape/incomplete_package` (integration)
- The pre-check stays bounded by the attempt budget: the model invariants InterceptsFollowRefusals, BudgetUnchanged, PendingOnlyAfterCompleteness, IncompleteBudgetEndsBlocked and TurnsBounded (unit)
- A complete first submission takes the ordinary path: acceptance "A complete first submission is accepted with no recovery and no extra request" (scenario)
- Refine, the recovery inference, the policy goldens and the force goldens are unchanged: existing `TestSubmissionRetryRefine`, `TestShapeRecoveryAfterReadinessFailures`, the policy golden tests and `TestShapeForceWireCompatibility` (existing)

## Feasibility
- Local scratch observation (a copy of the checkout outside it, not a Kogen probe): the harness compiles through the overlay against the unchanged tree, and every journey prints a KSR line. Unchanged-tree outcomes:
  - `missing-step-written` and `valid-first` already meet their scenarios.
  - `missing-step-prose` refuses the resubmission at attempt 2 with the `Unchanged:` line.
  - `intent-only` names only `acceptance.feature does not exist`.
  - `envelope-repaired` lacks the `received:` line.
  - `envelope-repeated` ends after 4 requests with `shape/invalid_output: the shaper's output was not a valid Intent after 3 attempts`.
  - A Cucumber dry run binds all 69 steps.
- The Quint model typechecks with Kogen's Quint. `quint run --backend=typescript --seed=7 --max-samples=5000 --max-steps=16` with the five invariants finds no violation. Each of these mutants violates an invariant: a pre-check that is never consumed; a budget end on a completeness refusal reported as invalid output; a bound of 10.
- The driver compiles a generated Go test and judges each journey with several assertions. Controls on the same submitted bytes qualify its judge:
  - `good` runs the live `valid-first` journey and judges known-good observations of the new behavior.
  - `bad` judges 15 known-bad observations and must reject every one. They include the unchanged tree's `missing-step-prose` and `envelope-repeated` observations, a missing or wrong `submission_precheck` event, an extra attempt, an extra request, the old envelope text and the old stop.
  - Observed in scratch: `good` printed `accepted` with exit 0; `bad` printed `rejected` with exit 1.
- evidence: {"id":"ksr-observer","kind":"observer","inputs":["features/steps/k-shape-recovery-completes-package__steps.js","features/steps/k-shape-recovery-completes-package__harness.js"],"observers":["features/steps/k-shape-recovery-completes-package__steps.js","features/steps/k-shape-recovery-completes-package__harness.js"],"checks":[{"name":"good","command":"\"$KOGEN_NODE\" features/steps/k-shape-recovery-completes-package__steps.js good","exit":0,"output":"accepted\n"},{"name":"bad","command":"\"$KOGEN_NODE\" features/steps/k-shape-recovery-completes-package__steps.js bad","exit":1,"output":"rejected\n"}]}
