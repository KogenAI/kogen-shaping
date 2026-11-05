# Return infrastructure errors from controlsRun

Slug: fix-k-shape-wrong-implementation
Model: not applicable — two local error-path fixes (controlsRun returns its own preparation and setup failures; contractCheck clears one field first) with no new state or interacting rules
Acceptance:
- .kogen/intents/fix-k-shape-wrong-implementation/acceptance.feature
- features/steps/fix-k-shape-wrong-implementation__steps.js
- features/steps/fix-k-shape-wrong-implementation__harness.js

An automated review of the landed k-shape-wrong-implementation commit found this problem. In `shape/controls.go`, `controlsRun` runs the candidate's acceptance once in a fresh copy outside the checkout. When it cannot prepare that copy, it returns `res, nil` instead of an error: creating the kit directory (`os.MkdirAll(kit, ...)`), writing the preload (`os.WriteFile(preload, ...)`), `sh.repo.Materialize(sh.ready.CandidateTree, tree)`, or a configured setup command that exits nonzero. The incomplete run then reaches `controlsUnreachable`, so the candidate is refused with `no pass is reachable: with the correct implementation in place, the run did not report every scenario as passed or failed`. A full disk or a failing `npm ci` therefore looks like a bad candidate and uses up the shaper's correction allowance. Separately, `contractCheck` (`shape/contract.go`) clears `sh.reviewProps` only after its model call. When it returns early, the properties of an earlier review stay behind. **Reproduced on the unchanged tree (HEAD b93f323)** with the harness of this Intent. A candidate tree that does not exist, TMPDIR set to a regular file, and setup `exit 3` each returned no error. Each produced both `no pass is reachable` lines and recorded a `refused` event after 1 run. After `contractCheck` failed reading a removed intent.md, `sh.reviewProps` still held 2 properties.

After this change:

1. `controlsRun` returns a non-nil error, and no result is judged, when any of these fails:
   - `os.MkdirAll` of its kit directory;
   - `os.WriteFile` of the preload;
   - `sh.repo.Materialize`;
   - a configured setup command that ran and exited with a nonzero status.

   For the first three, the error is the underlying error, wrapped or not. For setup, the error text contains `setup command "<cmd>" exited with status <n>`, as `fmt.Errorf("setup command %q exited with status %d", cmd, status)` writes it. Example: setup `exit 3` gives `setup command "exit 3" exited with status 3`.
2. The existing propagation carries that error unchanged: `controlsLines` → `wrongImplementationCheck` → `checkSubmission`. So Shape stops with the error, as it does for the check's other errors today. The check refuses nothing, starts no further run and, as for its other errors, records no `wrong_implementation_check` event.
3. These keep today's outcome (an incomplete run with no error, judged as now):
   - a failure to write the candidate's own controls.json files over the copy (`writeOverlay`: its symlink refusals and its directory and file writes), which comes from the candidate's controls.json;
   - an acceptance run that wrote no report;
   - a setup or acceptance command that the time cap cut off or left no time to start (the time-cap line);
   - an error from `sh.s.RunContext`, which is already returned.
4. `contractCheck` sets `sh.reviewProps = nil` as its first statement, before any early return. A contract check that returns before its review completes then leaves no properties behind, so a later `wrongImplementationCheck` finds no checked property and reads, runs and records nothing. The later `sh.reviewProps = nil` before parsing may stay or go. A complete review still sets `sh.reviewProps = props` as now.

Nothing else changes. The judging rules, refusal lines, caps, progress lines and events stay the same. So do Refine (whose `contractCheck` call also gets the reset, with no other effect), the prompts and the policy goldens. Every existing test stays green.

Build notes:
- The acceptance harness compiles `shape/fix_k_shape_wrong_implementation_acceptance_test.go` from the Go source in `features/steps/fix-k-shape-wrong-implementation__harness.js` through a Go overlay. Do not create that path, and do not use the `fkswi` or `FixKShapeWrongImplementation` identifier prefixes in the package.
- Keep these with their current signatures and behaviour, because the harness calls them:
  - in `shape/controls_test.go`: `controlsReal`, `controlsCandidate` (fields `feature`, `steps`, `correct`, `wrong`), `controlsCalcFeature`, `controlsExactDef`, `controlsCalc` and `controlsWrongs`;
  - `shpFx` fields `Root` and `Stderr`, and the constant `shpSlug`;
  - the `shaping` fields `ready` (`CandidateTree`), `cfg` (`Setup`) and `reviewProps`;
  - `(*shaping).contractCheck(ctx, hash, observations)` and `(*shaping).wrongImplementationCheck(ctx, hash)`, which returns `(contractConfirmed, error)` with field `wrong`.
- Acceptance scenario 6 runs 22 existing top-level tests by name with `-test.v` and requires each to pass, with no failed or skipped test or subtest:
  - TestParseControls, TestControlsJudge, TestControlsJudgeAll, TestControlsUnreachable, TestControlsRefusalJoinsContractRefusal and TestControlsPreloadFreezesAssert;
  - TestWrongImplementationRealRuns, TestWrongImplementationCaps and TestWrongImplementationCheckRecordsAndSkips;
  - TestContractEvidenceRule, TestContractPayloadValidation, TestContractCheckLastRefusalByAnotherGateWritesNoRecord, TestContractVerdictCounting, TestContractFormats and TestContractEnvironmentEvidenceRule;
  - TestContractCheckNumbersPropertyProblemsAfterContradictions, TestContractCheckRefusesAConfirmedContradictionAndPublishesItsCorrection, TestContractCheckKeptContradictionEndsNotReadyAndApprovalRefusesIt, TestContractCheckExternalShape, TestContractCheckEnvironmentFindings, TestContractCheckUnevidencedAndRefutedFindingsNeverRefuse and TestContractCheckReadOnlyToolRoundsAndIncompleteReviews.

  Keep their names. Add the new regression tests as new top-level test functions, not as subtests of these.
- The package's tests run in parallel. A Go test that changes TMPDIR must use the existing `isolatedTmp` helper.

## Decisions
- The wrong-implementation check returns an error, instead of refusing the candidate, when Kogen cannot prepare its copy (kit directory, preload, materialized candidate tree) or a configured setup command fails in it. (source: request)
- `contractCheck` clears the review properties first, so an early return leaves none of an earlier review. (source: request)
- A failure writing the candidate's own controls.json files over the copy is the candidate's problem and stays a refusal, because the request lists MkdirAll, WriteFile, Materialize and setup in `controlsRun`, not the overlay of controls.json. (source: request)
- The setup error names the command and its exit status: `setup command "<cmd>" exited with status <n>`. (source: request)
- Every existing test stays green, with no unrelated changes. (source: request)

## Source comparison
- Addressed: "Failures of MkdirAll, WriteFile, Materialize or setup return `res,nil`. The caller reports \"no pass reachable\"". Change 1. Scenarios 1 to 3 cover Materialize (a candidate tree that does not exist), the kit directory MkdirAll (TMPDIR a regular file, standing in for a full disk) and setup (`exit 3`, standing in for a failing `npm ci`). Each requires an error, no refusal, 1 run and no event. The unchanged tree shows each refused with "no pass is reachable".
- Addressed: "`sh.reviewProps` can also be stale on early returns" / "Reset `reviewProps` at the top of `contractCheck`". Change 4. Scenario 5 requires 0 properties after a contract check that failed reading intent.md, and then a wrong-implementation check with no error, refusal, run or event. The unchanged tree keeps 2 properties.
- Addressed: "Return infrastructure errors". Changes 1 and 2. The preload WriteFile failure gets the same treatment as the kit MkdirAll and has no separate scenario.
- Decided: the controls.json overlay (`writeOverlay`) and a missing acceptance report stay refusals, as today (change 3). Scenario 4 (a correct implementation path `calc.js/inner.js` beneath a file) requires today's two "no pass is reachable" lines, no error and a `refused` event. A version that returned every error would fail it.
- Addressed: "First reproduce it: add a test that fails on current main for exactly this scenario". The harness reproduced it on b93f323 (see the prose). Tests adds Go regression tests that fail on the unfixed code.
- Preserved: "Keep every existing test green; no unrelated changes". Scenario 6 runs, with the same test binary, the 22 existing tests of the two changed functions (named in the Build notes). It requires each to pass exactly once with no failed or skipped test or subtest. The configured `go-test` check runs the rest of the suite.

## Tests
- Failures to prepare the copy (kit directory, preload, Materialize) return an error from the wrong-implementation check instead of a "no pass is reachable" refusal: a new test in `shape/controls_test.go` using `controlsReal` with a candidate tree that does not exist, which fails on the unfixed code; acceptance scenarios 1 and 2 (regression)
- A setup command that exits nonzero returns the error `setup command "exit 3" exited with status 3` and refuses nothing: a new test in `shape/controls_test.go` with setup `exit 3`, which fails on the unfixed code; acceptance scenario 3 (regression)
- A failure writing the candidate's controls.json files over the copy is still refused with "no pass is reachable" and no error: acceptance scenario 4 (integration)
- A contract check that returns before its review leaves no review properties, and the following wrong-implementation check runs nothing: a new test in `shape/contract_test.go` or `shape/controls_test.go` that sets `reviewProps`, removes intent.md, calls `contractCheck` and expects nil `reviewProps`, which fails on the unfixed code; acceptance scenario 5 (regression)
- Existing tests stay green, including the judging, caps, refusal lines, progress and events of complete runs: the 22 existing controls and contract tests through acceptance scenario 6, and the whole suite through the configured go-test check (existing)

## Feasibility
- These are scratch runs from this read-only external session, not Kogen probe records. With go1.27.2 and Node v24.21.0, the harness Go source of `features/steps/fix-k-shape-wrong-implementation__harness.js` compiled into the shape tests through `go test -c -vet=off -overlay` (the build waited about 5 minutes at 1% CPU, apparently on a shared build cache). `TestFixKShapeWrongImplementationAcceptance` then passed in 1.7 s on the unchanged tree. It observed, for materialize, kit and setup, `error: ""`, both `no pass is reachable ... did not report every scenario` lines, 1 run and events `["refused"]`. For overlay it observed the same. For stale it observed `contract_error: true`, `review_properties: 2` and `error: "intent/parse: the Intent cannot be read"`. That is the reproduced fault; the feature expects the fixed form, and overlay as today.
- The same binary, run in `shape/` with `-test.run '^(<the 22 names>)$' -test.count=1 -test.v` and KOGEN_CUCUMBER set, exited 0 in 25 s on the unchanged tree. Each of the 22 tests printed one `--- PASS: <name> (` line, the real-run subtests included, with no FAIL or SKIP line. The step file's KEPT control input is those real result lines.
- The observer controls run the step file's judgments on known observations:
  - Good: the real `-test.v` output of the harness run with the values of every Then step of the feature, and the real result lines of the existing-test run.
  - Bad, for the harness: the unchanged tree's observed cases; failed, unfinished, failed-test, skipped, doubled, empty and observation-less runs; a missing case; a wrong or missing value for every compared key; a setup error without the command and status; and an overlay failure returned as an error.
  - Bad, for the existing tests: a failed or unfinished run, a failed, missing or doubled test, a failed or skipped subtest, and a run that ran no test.
- evidence: {"id":"infrastructure-error-observer","kind":"observer","inputs":["features/steps/fix-k-shape-wrong-implementation__steps.js","features/steps/fix-k-shape-wrong-implementation__harness.js",".kogen/intents/fix-k-shape-wrong-implementation/acceptance.feature"],"observers":["features/steps/fix-k-shape-wrong-implementation__steps.js"],"checks":[{"name":"good","command":"\"$KOGEN_NODE\" features/steps/fix-k-shape-wrong-implementation__steps.js --control good","exit":0,"output":"accepted\n"},{"name":"bad","command":"\"$KOGEN_NODE\" features/steps/fix-k-shape-wrong-implementation__steps.js --control bad","exit":1,"output":"rejected\n"}]}
