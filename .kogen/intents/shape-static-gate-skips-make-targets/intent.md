# Skip whole-project make targets in the static gate

Slug: shape-static-gate-skips-make-targets
Model: not applicable — recognising a command line as a whole-project check is a pure function of the text, with no state
Acceptance:
- .kogen/intents/shape-static-gate-skips-make-targets/acceptance.feature
- features/steps/shape-static-gate-skips-make-targets__steps.js

Shape's static gate never runs a whole-project check: Shaping checks the Intent, its steps and the files Shaping creates, while Build and landing run the full project suite (owner decision of 19 Nov, Studio document owner-decisions-20261119). The gate already skips a configured check whose command runs a test suite (status `not_run`, progress line `kogen: static check <name> not run: it runs the test suite`, outcome `none` when no other check remains). It does not recognise make targets, so ElixirDrops' check `spec`, which runs `make spec` (a project-wide Quint verification of about 200 s against the 120 s step limit of one check), is selected and run in a fresh copy every round.

A configured command line that runs `make` or `gmake` with any of the targets `test`, `tests`, `spec`, `specs`, `check`, `ci` or `verify` is a test suite check and is treated exactly as one. Recognition covers option words (words that start with a dash, for example `-j4`), a directory option (`-C` then a directory, whose value is never read as a target), other targets before or after, env assignments in front, and a `cd` prefix. The target must be a whole word followed by end of text, whitespace, a shell separator or a closing bracket, the same boundary rule as the existing pattern. Other make targets stay selected and run: `make lint`, `make fmt`, `make specimen`, `make test-unit` and `make -C spec lint`. Checks that are not skipped keep running unchanged and keep their attribution to the shaped files by path.

The change is one conservative extension of the existing suite pattern `testSuiteCheck` in `shape/static.go`, plus tests in `shape/static_test.go` (rows in the `TestStaticCheckSelection` table and one gate test). It adds no configuration, field or refusal text and changes no other check; the skip line text stays exactly as it is. Docs, README, the contract folder and every other repository are out of scope. The tests are model-free: no make, no Quint, no whole-suite run and no Studio tooling.

## Decisions
- Extend the existing `testSuiteCheck` pattern with one make alternative instead of adding a second pattern, function, setting or message; the existing leading and trailing boundary classes stay unchanged. (decision: D1, round 1) (constraint: existing boundary classes unchanged) (constraint: skip line text unchanged) (evidence: shape/static.go) (source: request)
- The make alternative is `make` or `gmake` after the leading boundary, then any number of words (option words starting with a dash, other non-dash words, or `-C` followed by one directory word, with a bare `-C` never read as an option word so its directory value cannot become the target), then one of `test`, `tests`, `spec`, `specs`, `check`, `ci`, `verify` as a whole word. (decision: D2, round 1) (evidence: shape/static.go; shape/static_test.go) (assumption: forms such as `make -f Makefile spec` match only because Makefile is read as another word; absolute paths before make and `--directory app` are not specifically required) (source: request)
- The gate (`staticGateRunner.run`) already handles a suite-recognised check as not_run and keeps attributing the others by path, so only the recognition changes; the gate test records every command given to the `staticRunCommand` seam. (decision: D3, round 1) (evidence: shape/static.go; shape/static_test.go) (source: request)

## Source comparison
- Treat `make spec`, `make test`, `make -j4 check`, `cd app && make verify`, `MIX_ENV=test make spec`, `gmake ci` and `make -C app tests` as test suite checks: addressed by the extended pattern and seven true rows (scenario "Whole-project make targets are test suite checks").
- Keep `make lint`, `make fmt`, `make specimen`, `make test-unit` and `make -C spec lint` selected: addressed by five false rows (scenario "Other make targets are not test suite checks").
- Gate with checks `spec` (`make spec`) and `vet`: only the vet command reaches the seam, spec is not_run with the unchanged progress line, vet is attributable to the shaped file, outcome refused: addressed (scenarios "The gate does not hand the make target to the command seam" and "The gate still judges the shaped files with the other check").
- Earlier lines `go test ./...`, `cd app && npm test`, `pytest -q`, `bin/rails test`, `bundle exec rake test`, `make && (cargo test)` stay true and `go build ./...`, `go testdata`, `bin/rails routes`, `rake testdata` stay false: addressed (scenario "Previously recognised commands keep their answers").
- Scope limited to `shape/static.go` and `shape/static_test.go`; no new configuration, field or refusal text; docs, README, contract folder untouched.

## Tests
- Make targets are recognised as whole-project checks and look-alike make commands are not: add the seven true and five false command lines as rows of TestStaticCheckSelection (unit)
- The gate does not hand `make spec` to the command seam, marks it not_run with the unchanged progress line, and still attributes and refuses on the other check: add TestStaticChecksSkipMakeSuite in shape/static_test.go (integration)
- Earlier recognised and not-suite commands keep their answers: the existing rows of TestStaticCheckSelection (existing)
- The recognition and the gate behave as specified: acceptance.feature, run through a Go overlay test of the shape package (scenario)

## Feasibility
- none
