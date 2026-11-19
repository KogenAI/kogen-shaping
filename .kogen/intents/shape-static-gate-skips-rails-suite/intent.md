# Skip Rails test suites in the static gate

Slug: shape-static-gate-skips-rails-suite
Model: not applicable — recognising a command line as a test suite is a pure function of the text, with no state
Acceptance:
- .kogen/intents/shape-static-gate-skips-rails-suite/acceptance.feature
- features/steps/shape-static-gate-skips-rails-suite__steps.js

Shape's static gate never runs a configured check that runs a whole test suite, but it does not recognise the Ruby on Rails suite commands, so on Rails projects it runs the full application suite in a fresh copy on every shaping round. The gate must treat a configured command line containing `bin/rails test`, `rails test` (also with a path, or a namespace such as `test:system`), `rake test` or `bin/rake test` (also behind `bundle exec` or environment assignments, and with a `test:` namespace) as a test suite check. Such a check then gets status not_run, prints `kogen: static check test not run: it runs the test suite`, and with no other check the outcome is none, exactly as for `go test`, `npm test` or `pytest` today. Recognition uses the same word-boundary rule as the existing pattern: the command word must be followed by end of text, whitespace, a shell separator or a closing bracket.

The change is one conservative extension of the existing suite pattern `testSuiteCheck` in `shape/static.go`, plus a focused model-free regression test that extends the table in `TestStaticCheckSelection` in `shape/static_test.go`. It adds no configuration and changes no other check, refusal text or event field. Commands that are not suites stay selected and run: `bin/rails routes`, `bin/rubocop`, `rake testdata` and `echo rails testing`. The commands recognised before (go test, cargo test, npm test, pytest and the others in the current pattern) stay recognised. Documentation, README and the contract folder are out of scope; no Rails, browser or Studio tooling and no whole-suite run is involved.

## Decisions
- Extend the existing `testSuiteCheck` pattern with a Rails alternative rather than adding a second pattern, setting or function; the existing leading and trailing boundary classes stay unchanged. (decision: D1, round 1) (evidence: shape/static.go) (source: request)
- The Rails alternative accepts the program words `rails` and `rake`, optionally preceded by `./` and `bin/`, followed by whitespace and `test`, optionally followed by a `:namespace` suffix; because it sits inside the existing boundaries, `bundle exec`, environment assignments, `cd app &&` and a trailing path argument need no extra handling, and `rails testing` and `rake testdata` do not match. (decision: D2, round 1) (evidence: shape/static.go; shape/static_test.go) (assumption: forms such as an absolute path before `bin/rails` or `rake -f Rakefile test` are not required) (source: request)
- The regression test extends the existing `TestStaticCheckSelection` table with the five Rails suite commands as true and the four look-alike commands as false, keeping every earlier row. (decision: D3, round 1) (evidence: shape/static_test.go) (source: request)

## Source comparison
- Recognise `bin/rails test`, `rails test` with a path or `test:` namespace, `rake test` and `bin/rake test`, also behind `bundle exec` or env assignments: addressed by the extended pattern and the five true rows (scenario "Rails suite commands are test suite checks").
- Keep `bin/rails routes`, `bin/rubocop`, `rake testdata`, `echo rails testing` selected: addressed by four false rows (scenario "Other Rails and rake commands are not test suite checks").
- Keep `go test ./...`, `cd app && npm test`, `pytest -q`, `go build ./...`, `go testdata` unchanged: addressed (scenario "Previously recognised commands keep their answers").
- Status not_run, the printed line `kogen: static check test not run: it runs the test suite` and outcome none follow from the unchanged suite-check handling in the gate; omitted as new behavior.
- Scope limited to `shape/static.go` and `shape/static_test.go`; docs, README and contract folder untouched.

## Tests
- Rails suite commands are recognised as test suite checks: add the five Rails commands as true rows to TestStaticCheckSelection (unit)
- Look-alike commands stay selected: add `bin/rails routes`, `bin/rubocop`, `rake testdata` and `echo rails testing` as false rows to TestStaticCheckSelection (unit)
- Earlier recognised and not-suite commands keep their answers: the existing rows of TestStaticCheckSelection (existing)
- The three groups of commands are answered by isTestSuiteCheck as specified: acceptance.feature, run through a Go overlay test of the shape package (scenario)

## Feasibility
- none
