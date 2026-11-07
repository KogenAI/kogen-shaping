# Name decision lines and controls before submit

Slug: k-shape-decision-lines-and-controls-named
Model: not applicable — guidance and refusal copy only; gate semantics remain unchanged
Acceptance:
- .kogen/intents/k-shape-decision-lines-and-controls-named/acceptance.feature
- features/steps/k-shape-decision-lines-and-controls-named__steps.js

Apply after k-shape-prompt-states-the-forms-its-gates-require. Update shape/prompts/shaper.md, shape/external.go, shape/controls.go and deliberate policy goldens under shape/testdata. Preserve existing gate outcomes and limits.

## Decisions
- Name required forms before the first submission. (source: request)
- Keep wrong-implementation controls distinct from Feasibility observer controls. (source: request)

## Required output
The shaper prompt says, before the first submission, that a submission's decisions are recorded as D<n> (numbered on from the decisions already recorded) of the current round, and that the same submission's Intent keeps each in ## Decisions as exactly its line: `- <decision> (decision: D<n>, round <r>)`, then ` (constraint: <c>)` per constraint, ` (evidence: <items joined by "; ">)`, ` (assumption: <uncertainty>)`, ` (revises: <id> because <why>)` when present, then ` (source: request)`. Decision evidence is files tracked at HEAD the shaping read (or a `ran` command of a probe of this shaping that succeeded), never a file the package adds. Omit absent optional markers, matching the actual ledger.

The note Kogen gives an external coding-agent session says a command that session ran is no evidence of a submitted decision; the files read are cited and what the command showed goes into the decision's uncertainty.

The shaper prompt names `.kogen/intents/<slug>/controls.json`: the last check needs it when the request asks for properties a step can observe, and it goes with the first package. Show its form: `{"correct":{"files":{"path":"whole content"}},"wrong":[{"name":"one property wrong","scenario":"scenario name","step":"Then exact feature text","files":{"path":"whole replacement content"}}]}`. A correct implementation makes every scenario pass; wrong implementations over it each get one property wrong, naming a scenario and the step line that must reject it, copied as the feature writes it, keyword included. State the limits: at most 7 wrong implementations, 8 files and 65536 content bytes per implementation, no acceptance machinery, all runs within the existing 300-second total check cap. Kogen runs them; they are not Feasibility's observer controls.

Gate text only: when a wrong implementation's step equals the text of a step of its scenario without the keyword, the refusal line names the keyword form (`write the step line as acceptance.feature writes it, keyword included: "<keyword> <text>"`). No gate accepts or refuses anything it does not today; the prompt goldens are updated with the prompt. A step belonging only to another scenario gets no hint.

## Source comparison
- Addressed Need 1 in Required output: D<n>, round, exact ordered ledger markers, same-submission carriage and tracked/read evidence restriction.
- Addressed Need 2 in Required output: external commands remain invalid evidence; cite files and describe findings as uncertainty.
- Addressed Need 3 in Required output: first-package controls.json, actual JSON keys, keyword line, correct/wrong overlays, limits, execution and distinction from observer controls.
- Addressed Need 4 in Required output: hint only for keywordless text in the named scenario; unchanged gate outcomes and updated goldens.
- Addressed Synthetic acceptance in Tests: include all supplied positive and negative gate cases, rendered guidance checks, and request-only synthetic multi-file web-application shaping with no answers. No application Build is requested.
- Superseded: discovering required forms through refusals. Omitted: no requested behavior. Do not add validation rules or relax evidence requirements.

## Tests
- Tracked/read evidence and exact carried ledger lines pass decision recording and carriage in one files submission: extend decision_record_test.go integration fixture, including multiple constraints and numbering after existing decisions (integration)
- Missing lines and one dropped constraint refuse; commands/prose from an external session and package-added files remain refused: preserve and extend decision record and carriage regressions (regression)
- Keyword-bearing controls parse; keywordless same-scenario text refuses with the keyword hint; another-scenario step refuses without hint: extend TestParseControls with these cases, preserving all existing outcomes (regression)
- Rendered shaper guidance includes ledger form, evidence rule and controls item; external note includes command-evidence sentence: text acceptance plus rendered-policy assertions and deliberately regenerated prompt goldens; never hand-merge hashes (integration)
- Synthetic multi-file web-application change in one request, shape only, request only, no answers: exercise existing scripted-provider Shape fixtures and assert first package contains controls.json and no refusal by carried-decisions gate, decision record for command evidence, or controls step missing keyword; retain explicit distinction between scripted plumbing and live-model compliance (integration)

## Feasibility
- none
