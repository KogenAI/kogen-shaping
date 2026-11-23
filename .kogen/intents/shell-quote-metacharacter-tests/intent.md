# Add shell quoting tests for hostile values

Slug: shell-quote-metacharacter-tests
Model: not applicable — the change only adds tests of a stateless function
Acceptance:
- .kogen/intents/shell-quote-metacharacter-tests/acceptance.feature
- features/steps/shell-quote-metacharacter-tests__steps.js

This change adds focused Go tests to `shape/context_test.go` that pin the behavior of `ShellQuote` in `shape/common.go`: it quotes a value for `/bin/sh` so that the shell reads the value back as exactly one literal argument. `ShellQuote` is not meant to change; its current behavior is the behavior to pin. The new tests evaluate the quoted text with `/bin/sh` itself. Three rules are pinned. An empty value, a value with spaces and a value with embedded apostrophes each give one argument equal to the original. A value with command substitution syntax and other shell metacharacters stays literal and executes nothing. A value with line breaks stays one argument with the line breaks preserved. The existing assertion on `it's` stays. Out of scope: shell command construction and the callers of `ShellQuote`.

## Decisions
- The tests live in `shape/context_test.go` and evaluate `ShellQuote(value)` with `/bin/sh`, then compare the number of arguments and the argument with the original value. Acceptance finds them at run time as the test functions of that file, other than the existing `TestShapeCommonHelpers`, whose body calls `ShellQuote(`, so their names are free. (evidence: shape/common.go; shape/context_test.go) (source: request)
- Empty input, spaces and embedded apostrophes each produce one argument equal to the original value when evaluated by /bin/sh. (evidence: shape/common.go; shape/context_test.go) (source: request)
- Input containing command substitution syntax remains literal when evaluated by /bin/sh and does not execute the substitution. The tests check that a marker file the substitution would create does not exist. (evidence: shape/common.go) (source: request)
- Input containing a line break remains one argument with the line break preserved. (evidence: shape/common.go) (source: request)
- `shape/common.go` keeps its `ShellQuote` unchanged; the change is test-only. Acceptance proves each test is meaningful by running the new tests in a scratch copy of the `shape` package against the unchanged `ShellQuote` (must pass) and against wrong versions (empty value gives no argument, spaces split, apostrophe escaped with a backslash, apostrophe doubled, double quotes leaving shell syntax active, quote ending at a line break), each of which at least one new test must reject. (evidence: shape/common.go) (source: request)
- The tests run `/bin/sh` directly and need no new dependency or helper outside `shape/context_test.go`. (evidence: shape/context_test.go; shape/helper_test.go) (source: request)

## Source comparison
- Request item 1 (empty input, spaces, apostrophes each one equal argument; malformed escaping is wrong) — addressed by the scenarios on an empty value, spaces, a backslash-escaped apostrophe and a doubled apostrophe.
- Request item 2 (command substitution stays literal; active shell syntax is wrong) — addressed by the scenario on double quotes that leave command substitution active.
- Request item 3 (line break stays one argument, preserved; ending or splitting at a line break is wrong) — addressed by the scenario on a quote that ends at a line break.
- Out of scope and omitted: shell command construction and callers of ShellQuote.

## Tests
- Empty input, spaces and embedded apostrophes each give one argument equal to the original under /bin/sh: a test in shape/context_test.go (unit)
- Command substitution syntax stays literal and executes nothing under /bin/sh: a test in shape/context_test.go (unit)
- A line break stays inside one argument, preserved, under /bin/sh: a test in shape/context_test.go (unit)
- The tests pass on the real ShellQuote and fail on each wrong ShellQuote: acceptance.feature scenarios run them in a scratch copy of the shape package (scenario)

## Feasibility
- none
