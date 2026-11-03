# Name ambiguous step definitions in self-check

Slug: k-shape-step-binding-proof
Model: not applicable — the step self-check is one static judgement of a candidate with no workflow state; only its feedback lines change
Acceptance:
- .kogen/intents/k-shape-step-binding-proof/acceptance.feature
- features/steps/k-shape-step-binding-proof__steps.js
- features/steps/k-shape-step-binding-proof__harness.js

Before publication, every concrete step line of a candidate's feature, including every expanded Scenario Outline example, must bind to exactly one step definition loaded the way the configured acceptance harness loads them. The existing step self-check (`shape/stepcheck.go` `stepSelfcheck`, using kogen-verification's `verify.StepAnalysis`, `verify.RunStepScript` and `verify.StepItems`) already does this. It loads the Intent's own step files and then the other JavaScript files of `features/steps/`, sorted, with Kogen's own Node and Cucumber. It counts matches with Cucumber's own `matchesStepName`, so Cucumber Expression/RegExp semantics and parameter types apply. It refuses zero or several matches, and a step file that does not load, before readiness and before any acceptance run. It never calls a step body or hook. Every distinct step text is counted, so an earlier failing step cannot hide a later unbound one. Keep that mechanism, its single `step_selfcheck` event and its unused-definition warnings. Do not add a second matcher. Do not infer matching from pattern-looking text. Do not escape a pattern again, rewrite a step or drop a scenario to satisfy the check.

Only the shaper feedback of `selfcheckScenarioFeedback` changes (the text returned to the shaper, the `step_selfcheck` refusal lines and the exhaustion diagnostic built from them):

1. An ambiguous step is reported at its feature location, with its step text and every matching definition. Today it is the location-less `- steps <text>: ambiguous step (<N> definitions)`. The new line is exactly
   `acceptance.feature line <L>: ambiguous step: <Keyword><text> matches <N> definitions: <pattern> in <file>; <pattern> in <file>`
   - `<L>` is the step's line in the feature.
   - `<Keyword>` keeps its trailing space, as in `Then `.
   - `<N>` is the existing count.
   - Each `<pattern> in <file>` pair is one definition that Cucumber's matcher matches for that text, in load order and separated by `; `. `<pattern>` is exactly the self-check's `StepDef.Pattern`: the RegExp source without slashes, or the expression string. `<file>` is its repository-relative step file.

   Take the matching definitions only from Cucumber's own matching through the existing script. For example, call `verify.RunStepScript` once more with the same files and only that one text: the definitions it marks `Used` are exactly those that match. Do this only when a step is ambiguous. kogen-verification is a pinned dependency and is not edited.
2. A step of an expanded Scenario Outline example adds ` (Examples row line <R>)` after the step text, in both undefined and ambiguous lines. `<R>` is that example's row line (`project.Scenario.Line` of a scenario whose `Outline` is set). An undefined Outline step reads `acceptance.feature line <L>: undefined step: <Keyword><text> (Examples row line <R>)`. There is one line per (step line, Examples row) instance. Steps of plain Scenarios keep today's exact `acceptance.feature line <L>: undefined step: <Keyword><text>` line, reported once per distinct step line even when several scenarios share it.

Unchanged:
- the headline `Step self-check failed; nothing was published. Fix the step files and resubmit.`;
- the `- load <file>: <error>` line and every other `- <check> <target>: <detail>` line;
- no binding results for a candidate whose own step file does not load;
- `step_selfcheck` event items, including the `ambiguous step (<N> definitions)` item detail;
- warnings, ownership rules, readiness and publication.

A passing binding check establishes binding only. It says nothing about whether a step body's observer is correct, and its feedback and events claim nothing more. No prompt, policy, tool definition, CLI flag or golden file changes.

## Decisions
- Binding is checked statically with the existing self-check and Cucumber's own matcher over every concrete and expanded line, without running bodies or hooks. Zero or several matches and load failures refuse before readiness. Unused definitions only warn. (source: request)
- Ambiguity feedback names the feature line, the step text and each matching definition's pattern and file. Undefined and ambiguous lines of expanded Outline examples also name the Examples row line. (source: request)
- The exact feedback line formats above extend the existing `acceptance.feature line <L>:` and `<pattern> in <file>` forms. (source: request)

## Source comparison
- Addressed: request Problem and Need. The existing static self-check is preserved, as the request asks; the probe below confirmed it already refuses zero matches, refuses load failures, checks expanded Outline lines and keeps unused warnings. The missing parts are ambiguity location and definitions and Examples-row locations (Rules 1 and 2).
- Addressed: request Synthetic acceptance items 1–6, with its complete feature and `features/steps/binding-proof__steps.js` literals. The harness keeps them byte for byte with one addition: its `neverRun` sentinel also appends to a trace file before it throws, so any executed body or hook is observable. The scripted shaper submits each case once through the real `Shape` entry point.
  - Item 1: the over-escaped `/^binding-proof: code \\d+$/` refuses at line 5.
  - Item 2: `/^binding-proof: code \d+$/` binds all five texts, both quoted values included.
  - Item 3: the `code {int}` duplicate names both patterns and their file. The repaired case is the "removing the duplicate" state.
  - Item 4: the deleted text definition leaves both Outline instances unbound at line 9, rows 12 and 13.
  - Item 5: `binding-proof: unused` gives one warning only.
  - Item 6: the top-level `throw new Error('cannot load definitions');` refuses with no binding results, no readiness record and no candidate acceptance run. The token pattern `/^binding-proof: token [a-f0-9]{8}$/` with `Then binding-proof: token a4b7c012` binds once.
- Addressed: "this is not a claim that the sentinel suite passes readiness". The passing cases assert only that the self-check passed and Shape went on to readiness, not that the package published.
- Addressed: "If all requirements already hold, close as verified coverage". The parts that already hold (items 1, 2, 5, 6 and the token case) are executable coverage and do not change at runtime. Only the two missing feedback details change.
- Omitted: the request's prohibitions (no re-escaping, rewriting or dropping) bind the implementation and add no automatic repair. Feature parse errors keep their existing route, where readiness refuses the unreadable package.

## Tests
- Ambiguous steps are refused at their feature line with the step text and every Cucumber-matched definition's pattern and file: update the "ambiguous step" case of `TestUnusedStepWarnShapeRefusals` to expect `acceptance.feature line 4: ambiguous step: Then is-even: it is even matches 2 definitions: is-even: it is even in features/steps/is-even__steps.js; is-even: it is even in features/steps/is-even__steps.js`, and update `TestSelfcheckFeedbackNamesDefectsNotWarnings` to the new ambiguous line (integration)
- Expanded Outline instances name their Examples row line in undefined and ambiguous lines, while plain Scenarios keep today's line: update `TestSelfcheckUndefinedOutlineRows` to use Outline-sourced scenarios and keep `TestSelfcheckUndefinedLocations` unchanged (unit)
- A RegExp that loads but is escaped twice matches nothing and is refused at each of its lines: add an "over-escaped RegExp" case to `TestUnusedStepWarnShapeRefusals` with `Given(/^is-even: the number \\d+$/, ...)`, expecting `acceptance.feature line 3: undefined step: Given is-even: the number 4` and `acceptance.feature line 6: undefined step: Given is-even: the number 3` (regression)
- Every concrete line and every expanded example binds once, or is refused before readiness, without executing step bodies or hooks: the binding-proof scenarios in acceptance.feature through the scripted shaper (scenario)
- Unused definitions only warn: `TestUnusedStepWarnShape` and the binding-proof "unused" scenario (existing)
- A step file that does not load refuses with its load error and no binding result: the "file that does not load" case of `TestUnusedStepWarnShapeRefusals` and the binding-proof "load-failure" scenario (existing)

## Feasibility
- A scratch overlay probe of the unchanged tree, outside the checkout, ran the six request cases through `Shape` with the scripted provider and the real Node 24.21.0 and Cucumber from testdata/tools/cucumber.
  - Already holds: the escaped RegExp gives `acceptance.feature line 5: undefined step: Then binding-proof: code 12` before readiness; the repaired, unused and token cases pass the self-check with all five texts bound once; the load failure gives `- load features/steps/binding-proof__steps.js: Error: cannot load definitions` with no binding items. No sentinel ran.
  - Does not hold yet: the duplicate gives only `- steps binding-proof: code 12: ambiguous step (2 definitions)`, and the Outline lines lack row numbers.
  - The draft acceptance in a scratch copy gave 5 passed and 2 failed (duplicate, text-deleted) on exactly those missing details. (probe: exit 0)
- evidence: {"id":"binding-proof-observer","kind":"observer","inputs":["features/steps/k-shape-step-binding-proof__steps.js","features/steps/k-shape-step-binding-proof__harness.js"],"observers":["features/steps/k-shape-step-binding-proof__steps.js"],"checks":[{"name":"good","command":"node features/steps/k-shape-step-binding-proof__steps.js good","exit":0,"output":"accepted\n"},{"name":"bad","command":"node features/steps/k-shape-step-binding-proof__steps.js bad","exit":1,"output":"rejected\n"}]}
