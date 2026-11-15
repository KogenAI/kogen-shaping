# Do not attribute differently failing static checks

Slug: shape-static-check-flaky-not-attributable
Model: not applicable — the change is one status decision in a single check run; there is no state to model
Acceptance:
- .kogen/intents/shape-static-check-flaky-not-attributable/acceptance.feature
- features/steps/shape-static-check-flaky-not-attributable__steps.js
- features/steps/shape-static-check-flaky-not-attributable__driver/harness_test.go.txt

The static gate (shape/static.go, `staticGateRunner.run`) runs each configured check on the shaped candidate, re-runs a failing one with `set -x`, runs it on the base tree without the shaped files, and today calls a failure attributable to the shaped files when the base run passed. A check such as a whole test suite can fail on the plain run and on the traced re-run with unrelated, different failures (a flaky test, or tests the shaped files cannot affect); today that refuses the submission with `static check test failed (exit 1) because of the shaped files: (not named by the check)`, which wastes a shaping round. The observed case: the plain run failed with 2 errors such as `NameError: undefined local variable or method 'saas'` in an unrelated test file, the traced re-run failed with one different failure, neither output named a shaped path (`paths: []`), and the same check passed on the next two submissions.

Now such a check has its own status `flaky` when all of these hold: the check failed on the candidate, the base run passed (so it would otherwise be attributable), the plain run's output names no shaped path, the traced re-run's output names no shaped path, and the sets of failing lines of the two runs differ. The failing lines of an output are its trimmed lines that contain `fail`, `error`, `panic` or `exception` (case-insensitive) and do not start with `+` (shell trace lines), with every run of decimal digits replaced by `#`, de-duplicated and sorted; only the set matters. A flaky check:
- leaves the run outcome `unknown` (like a pre-existing failure or an unavailable check) unless another check in the run is attributable, in which case the run is still `refused`;
- prints the progress line `kogen: static check <name> failed (exit <status>) with different failures on two runs of the same shaped files: not attributable; the shaped files are not established by this check` (no reused-base-result mark);
- is recorded in the `static_checks` event and the static record with `status` `flaky` and two new check fields, `failing` (the failing lines of the plain run) and `traced_failing` (those of the traced re-run), both omitted for every other status.

Unchanged: a failure that names a shaped path in either run, or that repeats with the same failing set in the traced re-run, is refused exactly as today; pre-existing and unavailable handling, the refusal text, the reuse text of approve and every other message. No setting is added.

A focused model-free regression test is added next to the other tests in shape/static_test.go.

## Decisions
- A flaky check gets the new status `staticFlaky` = `flaky`, counted with pre-existing and unavailable checks for the run outcome `unknown` (evidence: shape/static.go; shape/static_test.go) (source: request)
- The failing set of an output is its trimmed non-trace lines mentioning fail, error, panic or exception, digits normalised to `#`, de-duplicated and sorted; the request leaves "failing test names (or failure lines)" open and a generic line rule works for any test runner without parsing; digit normalisation keeps counts, timings and seeds from making identical failures look different (evidence: shape/static.go) (assumption: the operator accepts this generic line rule instead of runner-specific test-name parsing) (source: request)
- The recorded failing sets are the new check fields `failing` and `traced_failing`, set only on flaky checks and passed through the existing redaction and size bound line by line (evidence: shape/static.go) (source: request)
- The flaky decision applies only where the check would otherwise be attributable through a passing base run with no shaped path named; a check whose base run fails stays pre-existing, and a check that names a shaped path stays attributable (evidence: shape/static.go; shape/static_test.go) (source: request)
- Acceptance drives the real gate runner from a Go test harness copied into the shape package by a Go build overlay (an unwritten test is not run by name); a counter file makes the fixture check fail differently on each execution in the shaped copy and pass on the base tree (evidence: shape/static_test.go; shape/approve_test.go; features/steps/approve-static-reuse-line-explains-unknown__steps.js) (source: request)
- A flaky check gets the new status `flaky`, counted with pre-existing and unavailable checks for the run outcome `unknown` (decision: D1, round 1) (evidence: shape/static.go; shape/static_test.go) (source: request)
- The failing set of an output is its trimmed non-trace lines mentioning fail, error, panic or exception, digits normalised to `#`, de-duplicated and sorted (decision: D2, round 1) (evidence: shape/static.go) (assumption: The operator must accept this generic line rule instead of runner-specific test-name parsing; a scratch run showed the rule distinguishes the counter-file fixture but was not tried on real Rails output.) (source: request)
- Recorded failing sets are new check fields failing and traced_failing, set only on flaky checks (decision: D3, round 1) (evidence: shape/static.go) (source: request)
- Flaky applies only where the check would otherwise be attributable via a passing base run with no shaped path named in either run (decision: D4, round 1) (evidence: shape/static.go; shape/static_test.go) (source: request)
- Acceptance drives the real gate runner from a Go harness copied in by a Go overlay (decision: D5, round 1) (evidence: shape/static_test.go; shape/approve_test.go; features/steps/approve-static-reuse-line-explains-unknown__steps.js) (source: request)

## Source comparison
- Observed refusal, from the request: `static check test failed (exit 1) because of the shaped files: (not named by the check)` with `paths: []` and first-submission errors such as `NameError: undefined local variable or method 'saas'`; superseded for differently failing runs by the `flaky` status in scenario "A check failing differently on two runs without naming a shaped file is flaky and unknown".
- Different failures on two runs, no shaped path named, base passes: not attributable, status `flaky`, outcome `unknown`; addressed in the same scenario.
- Progress line text from the request, verbatim; and event fields recording status and both failing sets: addressed in "The flaky check says so in its progress line and records both failing sets".
- Failure repeating identically in the traced re-run is refused as today: addressed in "A failure that repeats identically without naming a shaped file is still refused".
- Failure naming a shaped path (in the plain run, or only in the traced re-run) is refused as today: addressed in "Failures that name a shaped file are still refused".
- Pre-existing handling, other messages, no new configuration: omitted by design (unchanged).
- Lab evidence (AoR cell h13 rails-ft-saas-billing baseline r3, whole Rails suite): context only; not reproducible here and replaced by the counter-file fixture the request describes.

## Tests
- A check that fails differently on its plain run and traced re-run, naming no shaped file, is `flaky`, the run outcome is `unknown`, and the progress line and the two failing sets are recorded: add a regression test in shape/static_test.go using a counter-file check script (regression)
- A check that fails identically twice, or names a shaped file, is still `attributable` and the run is `refused`: add to the same test and keep TestStaticChecksAttribution (regression)
- Pre-existing, unavailable and passing checks keep their status: TestStaticChecksAttribution and TestStaticBaseRunReused (existing)

## Feasibility
- none
