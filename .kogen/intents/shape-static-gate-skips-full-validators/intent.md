# Skip full-repository validators in the static gate

Slug: shape-static-gate-skips-full-validators
Model: not applicable — recognising a command line as a whole-repository check is a pure function of the text, with no state
Acceptance:
- .kogen/intents/shape-static-gate-skips-full-validators/acceptance.feature
- features/steps/shape-static-gate-skips-full-validators__steps.js

Shape's static gate never runs a full repository validator: Shaping checks the Intent, its steps and the files Shaping creates, while Build and landing run the full project suite (owner decision of 19 Nov, Studio document owner-decisions-20261119). The gate already skips a configured check whose command runs a test suite (status `not_run`, progress line `kogen: static check <name> not run: it runs the test suite`, outcome `none` when no other check remains). It does not recognise a script such as `python3 reproduce/validate_repo.py`, which kogen-bench configures as the check `validate-repo` and which takes 86 to 123 s against the 120 s step limit, so that validator is selected and run in a fresh copy every round.

A configured command line that runs a script whose file name is `validate_repo.py`, `validate_release.py` or `validate_all.py` is a whole-repository check and is treated exactly as a test suite check. The script may follow any interpreter word (python3, python, bash, sh) or none, have any directory part, a leading `./`, env assignments, a `cd` prefix and arguments such as `--strict`. The file name must be a whole word followed by end of text, whitespace, a shell separator or a closing bracket. Other scripts stay selected and run: `validate_round.py`, `validate_spec_snapshot.py`, `check_shaped_files.py`, `validate_repo.py.bak` and `python3 reproduce/validate_repo_helpers.py`. Checks that are not skipped keep running unchanged and keep their attribution to the shaped files by path.

The change is one conservative extension of the existing suite pattern `testSuiteCheck` in `shape/static.go`, plus tests in `shape/static_test.go` (rows in the `TestStaticCheckSelection` table and one gate test). It adds no configuration, field or refusal text and changes no other check; the skip line text stays exactly as it is. Docs, README, the contract folder and every other repository are out of scope. The tests are model-free: no validator run, no whole-suite run and no Studio tooling.

## Decisions
- Extend the existing `testSuiteCheck` pattern with one validator-script alternative instead of adding a second pattern, function, setting or message; the existing leading and trailing boundary classes stay unchanged, so the file name must be followed by end of text, whitespace, a shell separator or a closing bracket. (evidence: shape/static.go) (source: request)
- The alternative is an optional directory prefix (any run of characters other than whitespace, shell separators, parentheses and backquote, ending in `/`) followed by `validate_repo.py`, `validate_release.py` or `validate_all.py`; no interpreter word is required, so `./validate_repo.py` and `CI=1 bash ./validate_repo.py` match, while `validate_repo.py.bak`, `validate_repo_helpers.py`, `validate_round.py` and `validate_spec_snapshot.py` do not. (decision: D2, round 1) (evidence: shape/static.go; shape/static_test.go) (assumption: a bare script name with no interpreter, such as `./validate_repo.py`, may also be skipped) (source: request)
- The gate (`staticGateRunner.run`) already handles a suite-recognised check as not_run and keeps attributing the others by path, so only the recognition changes; the gate test records every command given to the `staticRunCommand` seam. (decision: D3, round 1) (evidence: shape/static.go; shape/static_test.go) (source: request)
- Extend the existing testSuiteCheck pattern with one validator-script alternative instead of adding a second pattern, function, setting or message. (decision: D1, round 1) (constraint: existing boundary classes unchanged) (constraint: skip line text unchanged) (evidence: shape/static.go; shape/static_test.go) (assumption: A bare script name with no interpreter (./validate_repo.py) is also skipped; the request allows any interpreter word and does not demand one.) (source: request)

## Source comparison
- Treat `python3 reproduce/validate_repo.py`, `python3 reproduce/validate_release.py --strict`, `cd reproduce && python validate_all.py` and `CI=1 bash ./validate_repo.py` as test suite checks: addressed by the extended pattern and four true rows (scenario "Whole-repository validator scripts are test suite checks").
- Keep `validate_round.py`, `validate_spec_snapshot.py`, `check_shaped_files.py`, `validate_repo.py.bak` and `validate_repo_helpers.py` selected: addressed by five false rows (scenario "Other validator-like scripts are not test suite checks").
- Gate with checks `validate-repo` and `privacy`: only the privacy command reaches the seam, validate-repo is not_run with the unchanged progress line, privacy is attributable to the shaped file, outcome refused: addressed (scenario "The gate does not run the validator but still judges the shaped files").
- Earlier rows `go test ./...`, `pytest -q`, `bin/rails test`, `make && (cargo test)` stay true and `go build ./...`, `bin/rails routes`, `echo pytester` stay false: addressed (scenario "Previously recognised commands keep their answers").
- Scope limited to `shape/static.go` and `shape/static_test.go`; no new configuration, field or refusal text.

## Tests
- Validator scripts are recognised as whole-repository checks and look-alike scripts are not: add the four true and five false command lines as rows of TestStaticCheckSelection (unit)
- The gate does not hand the validator to the command seam, marks it not_run with the unchanged progress line, and still attributes and refuses on the other check: add TestStaticChecksSkipFullValidator in shape/static_test.go (integration)
- Earlier recognised and not-suite commands keep their answers: the existing rows of TestStaticCheckSelection (existing)
- The recognition and the gate behave as specified: acceptance.feature, run through a Go overlay test of the shape package (scenario)

## Feasibility
- none
