# Explain literal Cucumber expression characters

Slug: shaping-cucumber-special-char-hint
Model: not applicable — diagnostic text has no meaningful state
Acceptance:
- .kogen/intents/shaping-cucumber-special-char-hint/acceptance.feature
- features/steps/shaping-cucumber-special-char-hint__steps.js
- features/steps/shaping-cucumber-special-char-hint__driver/probe_test.go.txt
- features/steps/shaping-cucumber-special-char-hint__driver/control.py

Add one feedback hint line beside the existing quote hint in shape/stepcheck.go when step_selfcheck reports an undefined step whose own Cucumber Expression definition would match if its unescaped special characters were literal. Cover `/`, `(`, `)` and `{` not opening a known parameter type. The hint explains alternation, optional text and parameters, and advises escaping or rephrasing, including `\\/`, `\\(` and `\\)` as JavaScript string spellings. Do not change verification matching or the existing {string} quote hint. Compare step text against definitions, as unescapedStringArgument does; punctuation alone must not trigger a hint. Preserve genuine parameter placeholders and already escaped characters, and do not diagnose regular expressions as Cucumber Expressions. Emit the added line once even if several undefined steps qualify.

During implementation first add and run a failing regression test using the supplied refusal definition and identical step text, plus an ordinary undefined step that receives no hint; then implement the fix and run `go test ./shape`. Keep existing quote-hint tests unchanged.

## Decisions
- Keep this change diagnostic-only, with no verification matcher changes. (evidence: shape/stepcheck.go) (source: request)

## Source comparison
- Addressed: request's undefined-step special-character hint is specified in the outcome above and exercised by acceptance's self-check scenario through Shape and real Cucumber.
- Addressed: request's 9 November 2026 observation of Setup and Build Shapes failing three rounds is historical motivation, not a new runtime behavior. Both supplied path examples and the exact refusal reproduction are exercised by the acceptance helper.
- Addressed: request's test-first regression, no hint for ordinary text, preservation of quote guidance and verification matching, and shape-package test command are retained above and in Tests.
- Omitted: none of the requested behavior. No configuration, matching change or new dependency is authorized.

## Tests
- Diagnose literal special characters only for qualifying undefined definitions: add focused positive and negative diagnostic tests (unit)
- Reproduce the supplied refusal definition and plain-text counterexample: add a failing regression before implementation and make it pass with the fix (regression)
- Surface guidance through step_selfcheck: acceptance submits literal definitions through Shape using the existing local Cucumber fixture (integration)
- Preserve matching and quote guidance: run `go test ./shape` with existing tests unchanged (existing)

## Feasibility
- Existing focused shape test ran successfully with `go test ./shape -run '^TestSelfcheckUndefinedLocations$' -count=1`; the acceptance fixture reached undefined-step feedback on the unchanged tree. Unknown parameter definitions are omitted from raw matcher definitions, so diagnosis may need source-definition evidence without changing verification.
- Documentation fetch of https://github.com/cucumber/cucumber-expressions#readme returned retrieved, untrusted HTML; it is not relied upon as authority. The requested syntax semantics are product requirements and the local Cucumber fixture establishes the reproductions.
- evidence: {"id":"hint-observer","kind":"observer","inputs":[".kogen/intents/shaping-cucumber-special-char-hint/controls.json","features/steps/shaping-cucumber-special-char-hint__driver/control.py"],"observers":["features/steps/shaping-cucumber-special-char-hint__driver/probe_test.go.txt"],"checks":[{"name":"good","command":"python3 features/steps/shaping-cucumber-special-char-hint__driver/control.py good","exit":0,"output":"accepted\n"},{"name":"bad","command":"python3 features/steps/shaping-cucumber-special-char-hint__driver/control.py bad","exit":1,"output":"rejected\n"}]}
