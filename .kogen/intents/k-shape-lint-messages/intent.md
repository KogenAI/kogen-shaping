# Expect the coordinator's current lint messages

Slug: k-shape-lint-messages
Model: not applicable — only the expected lint text in existing shape tests changes; kogen-shaping gains no state or product behaviour
Acceptance:
- .kogen/intents/k-shape-lint-messages/acceptance.feature
- features/steps/k_shape_lint_messages_steps.js
- features/steps/testdata/k_shape_lint_messages/recorded/pinned.stdout
- features/steps/testdata/k_shape_lint_messages/recorded/main_red.stdout
- features/steps/testdata/k_shape_lint_messages/recorded/main.stdout
- features/steps/testdata/k_shape_lint_messages/recorded/mutated.stdout

K-SHAPE-LINT-MESSAGES. kogen-coordinator main (8d24cc7, "Report every Intent lint problem in one refusal") deliberately replaced the single lint line `decisions: a decision has no source` with one specific line per unsourced Decision; the 3 Nov triage classes this INTENDED. kogen-shaping still pins an older coordinator (go.mod: `github.com/KogenAI/kogen-coordinator v0.0.0-20261102193107-c4a49a4fe84f`), and its shape tests quote the old line, so against coordinator main they fail and block every coordinator landing through the consumer check (k-coord-reject-max-rounds, 17:19Z).

The two coordinators' unsourced-Decision lint lines (observed, project/intent.go):
- pinned (c4a49a4): `decisions: a decision has no source` (once, however many Decisions lack a source);
- main (8d24cc7 and later; ffca707 = v0.0.0-20261103113131-ffca7072fe15, e26a4d7 changes documentation only): `decisions: <bullet> must end with `(source: request)` or `(source: answer <n>)``, one line per unsourced Decision, where `<bullet>` is the whole `## Decisions` bullet line including its leading `- `, for example `decisions: - Non-integers raise TypeError. must end with `(source: request)` or `(source: answer <n>)``.

After this change every shaping test that quotes the old line checks the lint correction with the coordinator's current decision-specific message and passes against both the pinned coordinator and coordinator main. Each such assertion is kept (never dropped or reduced to a prefix such as `decisions:` or a word such as `source`) and accepts exactly these two forms for its own unsourced Decision bullet: the pinned line, or main's line naming that bullet. A decisions line in any other wording fails it. The tests that quote the old line (the whole repository was searched; no fixture, golden or step file quotes it):
- `shape/contract_test.go:264`, TestShapeLintCorrectionsKeepEarlierArtifactsAndMalformedMixedPayloadsContributeNothing (subtests unsafe and duplicate): `anyContains(f.allToolOutputs(), "decision has no source")`, which fails with "the lint correction was not returned to the shaper". Bullet: `- Non-integers raise TypeError.`
- `shape/shape_test.go:504`, TestShapeInvalidSubmitIsRetriedWithFeedbackAndKeepsEarlierProposals: compares the whole feedback with `"invalid submission:\ndecisions: a decision has no source\nEarlier well-formed proposals are retained; submit only corrections."`. It keeps comparing the whole feedback: exactly those three lines, with the middle line in either form. Bullet: `- Non-integers raise TypeError.`
- `shape/refine_test.go:857`, TestRefineNarrowRetryPreservesAllEarlierProposalsUntilMergedReadiness: `shpContains(t, "lint feedback", lint, "decisions: a decision has no source")`. Bullet: `- Non-integers raise TypeError.` Its other assertions (`model: the Model: line must be`, the retention sentence) stay.
- `shape/refine_test.go:927`, TestRefineLaterProposalsReplaceAPathAndOriginalBytesRevertIt: `shpContains(t, "feedback", …, "decisions: a decision has no source")`. Bullet: `- Non-integers raise TypeError. Retain the feature correction.`

Only these `_test.go` files change; a small shared test helper in the `shape` package's test files is allowed. No product behaviour change: no non-test Go file, go.mod or go.sum changes, and go.mod keeps requiring the pinned coordinator. `shape/context_test.go:136` ("ordinary has no source") is unrelated and stays.

## Decisions
- The lint-correction assertion is kept and checks the coordinator's current decision-specific message, matching what both the pinned coordinator and coordinator main say about the unsourced Decision. (source: request)
- Every other shaping test or fixture that quotes the old single-line message follows the same way. (source: request)
- No product behaviour changes. (source: request)

## Source comparison
- addressed: request "match what both say about the unsourced Decision" and "Never drop the assertion"; contract: prose (two exact forms per bullet, other wording fails) and acceptance.feature (pinned and main pass; the mutated coordinator, whose line reads `decisions: <bullet> is unsourced`, makes every listed test fail).
- addressed: request TestShapeLintCorrectionsKeepEarlierArtifactsAndMalformedMixedPayloadsContributeNothing (shape/contract_test.go:265, subtests unsafe and duplicate); contract: prose list and acceptance.feature rows including both subtests.
- addressed: request "Any other shaping test or fixture that quotes the old single-line message"; contract: shape/shape_test.go:504, shape/refine_test.go:857 and :927 in prose and acceptance.feature rows.
- addressed: request "No product behaviour change"; contract: prose and the acceptance step that go.mod requires kogen-coordinator v0.0.0-20261102193107-c4a49a4fe84f without a replace.
- addressed: request coordinator main 8d24cc7; contract: acceptance runs ffca707 (v0.0.0-20261103113131-ffca7072fe15), whose code descends from 8d24cc7 with the same lint line; main's newer head e26a4d7 changes documentation only.

## Tests
- The lint-correction check of TestShapeLintCorrectionsKeepEarlierArtifactsAndMalformedMixedPayloadsContributeNothing accepts the pinned or main unsourced-Decision line and rejects other wording: update shape/contract_test.go:264 (regression)
- The whole-feedback check of TestShapeInvalidSubmitIsRetriedWithFeedbackAndKeepsEarlierProposals accepts either form of the middle line: update shape/shape_test.go:504 (regression)
- The lint feedback checks of TestRefineNarrowRetryPreservesAllEarlierProposalsUntilMergedReadiness and TestRefineLaterProposalsReplaceAPathAndOriginalBytesRevertIt accept either form for their own bullet: update shape/refine_test.go:857 and :927 (regression)
- No product behaviour change: the existing shape suite passes unchanged against the pinned coordinator (existing)

## Feasibility
- evidence: {"id":"lint-messages-observer","kind":"observer","inputs":["features/steps/k_shape_lint_messages_steps.js",".kogen/intents/k-shape-lint-messages/acceptance.feature","features/steps/testdata/k_shape_lint_messages/recorded/pinned.stdout","features/steps/testdata/k_shape_lint_messages/recorded/main_red.stdout","features/steps/testdata/k_shape_lint_messages/recorded/main.stdout","features/steps/testdata/k_shape_lint_messages/recorded/mutated.stdout","go.mod"],"observers":["features/steps/k_shape_lint_messages_steps.js"],"checks":[{"name":"good","command":"\"$KOGEN_NODE\" features/steps/k_shape_lint_messages_steps.js --control good","exit":0,"output":"accepted\n"},{"name":"bad","command":"\"$KOGEN_NODE\" features/steps/k_shape_lint_messages_steps.js --control bad","exit":1,"output":"rejected\n"}]}
- Scratch work in this read-only external session, outside the checkout and not a Kogen probe record (go1.27.2, Node v24.21.0, Kogen's Cucumber at $KOGEN_CUCUMBER). A blob-less clone of github.com/KogenAI/kogen-coordinator showed main at e26a4d7 with 8d24cc7 in its history; ffca707..e26a4d7 changes only docs and Intent files, and project/intent.go's decisions line is identical from 8d24cc7 on. ffca707 is in the local Go module cache and its go.mod equals the pinned one's.
- Unchanged tree: the four tests (and both subtests) pass against the pinned coordinator and fail against ffca707 at contract_test.go:265, shape_test.go:504, refine_test.go:857 and :927, with the main lines quoted in the prose. The acceptance command on an export of HEAD ran 1 scenario: 6 steps passed and the last (main) failed, in about 16 s.
- A scratch prototype of the test change (a helper accepting the two exact forms) passed all four tests against the pinned and ffca707 coordinators and failed each against the mutated copy; the acceptance command passed 7 of 7 steps in about 14 s. The recorded outputs are those real runs: pinned and main_red from HEAD, main and mutated from the prototype.
