# Report tracked directories in question evidence

Slug: shape-tracked-directory-evidence-no-slash
Model: not applicable — the change is a stateless message classification of a cited path
Acceptance:
- .kogen/intents/shape-tracked-directory-evidence-no-slash/acceptance.feature
- features/steps/shape-tracked-directory-evidence-no-slash__steps.js

Shape must use Git HEAD to recognize a cited tracked directory in either spelling (`docs/presets` and `docs/presets/`) wherever a citation is judged, never calling it untracked, while the stronger file-only requirements stay. The Decisions checker, the decision context and the sourced decision ledger already do this at HEAD. What remains is the Questions cited-evidence block, which still reports a tracked directory in either spelling as `is not a file tracked at HEAD`. That block must instead say the path is a directory and tell the shaper to cite the files it read, with the same wording the sourced decision ledger uses. Untracked and nonexistent paths keep the `is not a file tracked at HEAD` wording, unread tracked files and unlisted directories stay refused, and a listed tracked directory stays accepted in Decisions evidence in both spellings.

## Decisions
- Reuse the ledger wording "<path> is a directory; cite the files this shaping read" for a tracked directory cited in a question, keeping the spelling the shaper wrote. (evidence: shape/question.go; shape/ledger.go) (source: request)
- Leave the Decisions checker, the decision context and the ledger unchanged, because HEAD already recognizes both spellings and the regression test for them exists. (evidence: shape/evidencemark.go; shape/decision_evidence.go; shape/decision_evidence_grounding_test.go) (source: request)
- Keep file-only evidence in the ledger and questions: a directory is never shown as file content. (evidence: shape/ledger.go; shape/question.go) (source: request)
- Change only the Questions cited-evidence block to report a tracked directory in either spelling as a directory and ask for files. (decision: D1, round 1) (evidence: shape/question.go; shape/evidencemark.go; shape/decision_evidence.go; shape/ledger.go; shape/decision_evidence_grounding_test.go) (assumption: The Go harness was not run in this session; readiness will show whether it compiles.) (source: request)

## Source comparison
- Tracked directory recognized in either spelling by HEAD: already addressed in shape/evidencemark.go and shape/decision_evidence.go; acceptance re-observes it.
- File-only ledger calls the directory untracked: already says it is a directory; acceptance re-observes it. The Questions cited-evidence block still said untracked and is changed.
- Unread tracked file, unlisted directory and untracked directory stay refused: observed in acceptance.
- The older checker's rejection of a ten-line claim against eleven real lines: unrelated to this change and left as is.

## Tests
- A question citing a tracked directory in either spelling is told it is a directory and to cite files: a regression test of citedEvidence in the shape package (regression)
- A question citing an untracked directory keeps the untracked wording: the same regression test (regression)
- A listed tracked directory is accepted in Decisions in both spellings, an unread file and an unlisted directory are refused, an untracked directory is refused: the existing and acceptance-observed cases (existing)

## Feasibility
- none
