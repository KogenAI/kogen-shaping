# Refuse malformed shaping questions before asking

Slug: k-shape-question-options-recommended-checked
Model: not applicable — one stateless check of each questions submission; the retry bound and the stop are the existing invalid-submission counter and error, unchanged
Acceptance:
- .kogen/intents/k-shape-question-options-recommended-checked/acceptance.feature
- features/steps/k-shape-question-options-recommended-checked__steps.js
- features/steps/k-shape-question-options-recommended-checked__harness.js

Today Shape accepts a questions submission whose questions have no options, an empty option list, or options without a `recommended` mark. `validateQuestions`/`validOptions` in `shape/shape.go` treat options as optional, and the recommended mark is only looked for in `recommendedAnswers`, when the answers are expanded. So Shape prints such a question (`printQuestions`), stores it as pending (`state.json` and `questions.json` via `writeQuestions`) and exits 5. The user's `all: recommended` then fails with `intent/answers_invalid: question 1 has no recommended option` (exit 2), and the run cannot continue from that reply. Acceptance scenario 3 reproduces this on the unchanged tree.

After this change, every question that Shape prints, stores as pending or writes to `questions.json` has meaningful options and exactly one recommended option. This holds for every kind (product/UI-UX and handoff alike). Meaningful options means at least two options whose labels are non-empty and distinct.

**Where it is checked.** The check is part of the submission's validation (`validateQuestions`, reached from `validate` in `submitOne`). It applies to the native and the external shaper alike, since both submit through `sh.submit`. It runs before the decision record gate, the question check model request, `state.json`, `questions.json` and any printing. A submission with a malformed question is refused as a whole: none of its questions is asked, including well-formed ones. It never reaches the question check, so a malformed question costs no model request beyond the shaper's own next turn. A submission whose questions are all well-formed makes exactly the provider requests it makes today.

**The rule, per question.** `options` is the question's `options` value.
- Absent or JSON null: problem `no options`. Nothing else is reported for that question.
- An empty list: problem `an empty option list`. Nothing else is reported for that question.
- Otherwise, in this order:
  - `only one option` when the list has one option;
  - for each option in order (letters a, b, c, ...): `option <x> has an empty label` when its label is empty or only whitespace. Otherwise, `option <y> repeats the label of option <x>` when its label, with surrounding whitespace trimmed, equals the trimmed label of an earlier option `<x>` (the first such option). Comparison is exact and case-sensitive;
  - `no option is marked recommended` when no option has `recommended: true`, or `<k> options are marked recommended (<letters joined by ", ">)` when more than one does.

**Feedback.** Each problem is one line, `question <n> (<question text verbatim>): <problem>`, with questions in submission order (numbered from 1) and each question's problems in the order above. After all problem lines comes one rule line: `every question needs at least two options with distinct, non-empty labels and exactly one option marked recommended`. Those lines are the refusal's problems. They go through the existing validation refusal path:
- a `submission_refused` event with gate `validation`;
- one counted invalid submission;
- feedback to the shaper of exactly `invalid submission:` + newline + the lines joined by newlines + newline + `No proposals from this submission were retained; earlier proposals are unchanged.`

The exact texts are in acceptance scenarios 1–9.

**Precedence among validation problems.** The existing checks still come first, each as a whole-submission single line as today. The structural check comes first: question text, and options that are a list of objects with string labels and boolean `recommended` values. A non-list `options` value other than null stays structural. Its line changes from `every question needs question text and optional options with string labels and boolean recommended values` to `every question needs question text and options with string labels and boolean recommended values`. Next comes the rejected-kind line (`questions: decide technical questions inside shaping; only kind product or handoff may reach the caller`, unchanged). The option problems above apply only when both pass. Malformed `decisions` beside the questions are still reported as today, after these lines.

**Bound and stop.** No new bound is added. Refusals count toward the existing invalid-submission limit (`submissionRetryLimit`, default 3, `ShapeInput.SubmissionRetryLimit`). When it is reached, Shape ends with the existing `shape/invalid_output: the shaper's output was not a valid Intent after <N> attempts` (exit 1), or with the existing `intent/not_ready` when files were proposed earlier in the run. Its lines are the existing ones: the last refusal's problem lines, which name the malformed question, then the existing `earliest unresolved refusal (attempt <a>, gate validation):` block when it differs. Nothing is printed to stdout, and no `state.json` round or `questions.json` is written for the malformed question (scenario 9).

**Unchanged.**
- Which questions are asked: the question check, its classes and its instructions.
- The shaper's prompts, the `submit` tool definition and the policy goldens (`shaper_policy_golden.json`, `prompts_golden.json`, `defs_golden.json`).
- The printed question block, the `questions.json` envelope and fields, answer parsing, explicit answers and `all: recommended` for well-formed questions.
- No option wording is refused, so a handoff option that waits or pauses is asked (scenario 12). No answer is inferred and approval is not touched.
- Pending rounds that an earlier version already stored are not re-validated.

Because each asked question has exactly one recommended option, the printed `(recommended)` mark, the `questions.json` `recommended` letter and the option `all: recommended` picks are the same option.

**Harness.** The acceptance harness compiles into the shape package's tests through a Go overlay at `shape/k_shape_question_options_recommended_checked_acceptance_test.go`. That path must never exist in the repository. The harness uses these shape test helpers, whose names and signatures the Build keeps:
- `newShpFx`, `shpFx.setReport`, `shpFx.shape`, `shpFx.exists`, `shpFx.Root`, `shpFx.Stdout`, `shpFx.Prov`;
- `shpProvider.route`, `shpProvider.allRequests`;
- `shpSubmit`, `shpQuestionCheck`, `shpFilesWithDecisions`, `contractKind`, `decisionRequest`, `literalRequest`, `shpToolNames`, `shpOutputs`, `shpBody`;
- `SubmitQuestionCheckTool`.

## Decisions
- Every question that Shape prints, stores as pending or writes to questions.json has at least two options with distinct, non-empty labels and exactly one option marked recommended, for every question kind (source: request)
- The check runs where a questions submission is accepted, before the question check, printing or storing; a malformed question returns to the shaper as repairable feedback naming what is missing (source: request)
- Refusals use the existing invalid-submission limit and its existing `shape/invalid_output` stop, whose lines name the malformed question; no new bound and no extra model request (source: request)
- Which questions are asked, the option wording allowed, explicit answers and `all: recommended` for well-formed questions are unchanged (source: request)
- The feedback lines read `question <n> (<question text>): <problem>` followed by one rule line (assumption: the request names the problems but leaves their wording open) (source: request)
- A single option is malformed (only one option), and labels are compared after trimming surrounding whitespace, case-sensitively (assumption: the request defines meaningful as at least two distinct, non-empty labels without a comparison rule) (source: request)
- Validate question options inside validateQuestions (the submission's validation), before the decision record gate, the question check request, state.json, questions.json and printing; refusals take the existing validation refusal path and the existing invalid-submission limit and shape/invalid_output stop (decision: D1, round 1) (constraint: No new retry bound or error class) (constraint: A malformed question triggers no question check request) (constraint: Well-formed submissions keep today's provider request sequence) (evidence: shape/shape.go; shape/question.go; shape/shape_test.go) (source: request)
- Leave the shaper prompts, the submit tool definition and the policy goldens unchanged (decision: D2, round 1) (constraint: shaper_policy_golden.json, prompts_golden.json and defs_golden.json keep their bytes) (evidence: shape/policy.go; shape/prompts/shaper.md; README.md) (assumption: A model may still submit malformed questions; the feedback and the existing bound handle that) (source: request)
- Acceptance runs a Go harness overlaid into the shape package's tests with a request-kind-routed scripted question check, observing stdout, questions.json, state.json rounds, provider request kinds and shaper feedback; the step file carries good/bad observer controls grounded in real unchanged-tree observations (decision: D3, round 1) (constraint: shape/k_shape_question_options_recommended_checked_acceptance_test.go never exists in the repository) (constraint: The shape test helpers the harness uses keep their names and signatures) (evidence: features/steps/k-shape-environment-stop-evidence__steps.js; features/steps/k-shape-environment-stop-evidence__harness.js; shape/helper_test.go; shape/question_helper_test.go) (source: request)

## Source comparison
- addressed: Need, "every question printed, stored as pending or returned to a machine caller carries meaningful options and exactly one recommended option" — the per-question rule above, checked in `validateQuestions` for product and handoff kinds alike; scenarios 1–12.
- addressed: Need, "check this where a question submission is accepted, before anything is printed, stored or returned" — validation precedes the question check, `state.json`, `questions.json` and `printQuestions`; scenarios observe `request_kinds`, `state_rounds`, `questions_json` and stdout.
- addressed: Need, feedback names what is missing (no options, an empty list, an empty or repeated label, no recommended option, more than one) — scenarios 1 (`no options`), 2 (`an empty option list`), 3 (`no option is marked recommended`), 4 (`2 options are marked recommended (a, b)`), 5 (`option b repeats the label of option a`), 6 (`option b has an empty label`); scenario 7 covers "at least two options" (`only one option`); scenario 8 covers numbering within a multi-question submission.
- addressed: Need, existing retry bound, no new bound, no extra model request — scenario 9 (3 attempts, `shape/invalid_output`, no question check request); scenarios 11 and 12 pin today's request sequence for well-formed questions.
- addressed: Need, honest stop with that cause and no published malformed question — scenario 9 (error lines name question 1 and its problems; stdout empty; no `questions.json`, no `state.json`).
- addressed: Need, no change to which questions are asked, no classifier, no evidence condition, no refused option wording — the question check is untouched; scenario 12 asks a handoff whose recommended option waits.
- addressed: Need, no inferred answer, no secret solicited, approval not weakened — nothing in answers or approval changes; the scenarios' second runs record the user's own answer (`receipt_answers`).
- addressed: Need, explicit answers and `all: recommended` behave as today for well-formed questions — scenario 11 (`1: b`), scenarios 1–3 and 10 (`all: recommended`, exit 0, published).
- addressed: Need, text output and machine output show the same options and recommended option — scenarios 1, 10, 11 and 12 compare stdout with `questions_json`.
- addressed: Synthetic acceptance 1 — scenarios 1, 2 and 3 (one question three ways; nothing printed, stored or returned; feedback names the problem; the repaired question emitted once; scenario 3 is the reported `answers_invalid` reproduction).
- addressed: Synthetic acceptance 2 — scenarios 4 (two marked recommended), 5 (repeated label) and 6 (empty label).
- addressed: Synthetic acceptance 3 — scenario 9.
- addressed: Synthetic acceptance 4 — scenario 10 (handoff, (a) recommended, `all: recommended` selects (a), text and machine output alike, no answers_invalid).
- addressed: Synthetic acceptance 5 — scenario 11 (`request_kinds` equal to the unchanged tree's real sequence).
- addressed: Synthetic acceptance 6 — scenario 12 (handoff with a waiting option is asked); scenario 10's option b also pauses.
- addressed: invented repository, scripted shaper and question check, no live accounts, credentials or external services — the shape tests' is-even fixture and scripted local provider.
- superseded: existing unit and integration tests that submit option-less questions or handoffs without options as fixtures, and `TestShapeAllRecommendedNeedsARecommendedOptionForEveryQuestion`, which asserts the old post-hoc `answers_invalid` for a newly asked option-less question.

## Tests
- A question with absent/null options, an empty list, one option, an empty or repeated label, no recommended option or more than one is refused with the exact problem lines and rule line, in question and problem order; a well-formed product or handoff question passes: add a table test of `validateQuestions` in shape/discovery_test.go or shape/shape_test.go (unit)
- The structural validation line no longer says options are optional: update the expected text in `TestShapeSubmissionEnvelopeProblems` (unit)
- A malformed question is refused through the validation gate without a question check request, nothing is printed or stored, and the repaired question is asked and answered by all recommended: replace `TestShapeAllRecommendedNeedsARecommendedOptionForEveryQuestion` with a regression test of that journey in shape/shape_test.go (regression); scenarios 1–3 (scenario)
- Two recommended marks, a repeated label, an empty label and a single option are refused with feedback naming the problem: scenarios 4–8 (scenario)
- Malformed submissions stop at the existing limit with `shape/invalid_output`, lines naming the question, nothing pending: scenario 9 (scenario); `TestShapeThreeInvalidSubmissionsStopShapingAndWriteNothing` (existing)
- Well-formed handoff and UI/UX questions are asked as today with the same provider requests, explicit answers and all recommended continue: scenarios 10–12 (scenario); `TestShapeAllRecommendedAnswersEveryQuestion`, `TestShapeAnswersContinueShapingAndTheDecisionRecordsItsSource` (existing)
- Test fixtures that submit option-less product or handoff questions now meet the option rule: update `TestShapeQuestionsPathPrintsNumberedQuestionsAndWritesNothing`, `TestShapeQuestionsFileListsKindsOptionsAndRecommendations`, the handoff questions in shape/question_test.go and shape/discovery_test.go and the other `shpQuestion(text)` calls without options to submit well-formed questions, keeping what each test asserts (integration)

## Feasibility
- These are scratch runs from this read-only external session, in a `git archive` copy of HEAD 4a41927 outside the checkout. They are not Kogen probe records. The harness compiled through `go test -c -vet=off -overlay` in about 3 s, and each of its twelve cases passed the harness test in under 2 s on the unchanged tree. The baseline reproduces the reported fault. In the no-options, empty-options and no-recommended cases the malformed question is printed and written to `state.json` and `questions.json` with exit 5, and the second run's `all: recommended` ends with `intent/answers_invalid: question 1 has no recommended option` (exit 2). In two-recommended, the question is printed with two `(recommended)` marks while `questions.json` says `"recommended": "a"`. The handoff, ui and handoff-wait cases produced exactly the observations the feature expects. One focused Cucumber run in that copy passed the ui scenario and failed the no-options scenario at its first Then step (`refused_gates is [], want ["validation"]`).
- The observer controls below run the step file's judgments. The good control uses the real `-test.v` output of a harness run on the unchanged tree, each scenario's expected observation, and the real unchanged-tree observations of the three preserved cases. The bad control uses the real unchanged-tree observations of the nine malformed-question cases, plus a failed, timed-out, doubled or observation-less run, another case's observation, and single-field corruptions of every compared value.
- evidence: {"id":"question-options-observer","kind":"observer","inputs":["features/steps/k-shape-question-options-recommended-checked__steps.js","features/steps/k-shape-question-options-recommended-checked__harness.js",".kogen/intents/k-shape-question-options-recommended-checked/acceptance.feature"],"observers":["features/steps/k-shape-question-options-recommended-checked__steps.js"],"checks":[{"name":"good","command":"node features/steps/k-shape-question-options-recommended-checked__steps.js --control good","exit":0,"output":"accepted\n"},{"name":"bad","command":"node features/steps/k-shape-question-options-recommended-checked__steps.js --control bad","exit":1,"output":"rejected\n"}]}
