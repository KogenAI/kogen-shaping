# Return step-check defects for correction

Slug: k-shape-stepcheck-repair
Model: not applicable — wire step-code findings into the existing bounded correction loop without adding workflow state
Acceptance:
- .kogen/intents/k-shape-stepcheck-repair/acceptance.feature
- features/steps/k_stepcheck_repair_steps.js
- features/steps/k_stepcheck_repair_test.go.txt

A failed final step-code check must return actionable findings to the shaper as a correction, just like other readiness findings. Recheck the complete corrected package and publish only after it passes. Replace the premature terminal intent/not_ready outcome with the existing bounded correction route; only exhaustion returns intent/not_ready with the remaining findings. Keep existing correction limits, timeouts, CLI and readiness checks. No network tests or new flags/commands.

The reported first-user reproduction was a three-file Go module (Greet plus one test), request `Greet should return "Hello, world" when the name is empty`. Release v0.1.0-dev.9bf6073f, ChatGPT gpt-6.1-sol default lineup, stopped after 121 seconds and 8 requests at `kogen: checking step code (121 s)`. Its error was `intent/not_ready: the Intent is not ready to build` with finding `exit_code features/steps/empty-name.js: step definition I call Greet with name {string} asserts an exit code but obtains it through go run, which changes exit codes; build the binary (go build) and run it instead`. Preserve this actionable content, not the accidental elapsed time or provider request count. Tests must be synthetic fake-model tests. This does not change Greet or relax the checker; a corrected step builds and executes a binary rather than using go run to observe an exit status.

## Decisions
- Return concrete step-code findings for correction and publish only passing packages; report remaining findings at exhaustion. (source: request)
- No new CLI commands or flags, no increased timeout, and no network in synthetic fake-model tests. (source: request)

## Source comparison
- Addressed: request Need and Problem, including the go run/go build exit_code finding, by the correction and exhaustion journeys in acceptance.feature and the fake-provider overlay harness. Existing shape/checkSubmission and stepSelfcheck are evidence of a partial feedback route, not authority to declare the reported defect fixed.
- Addressed: request reproduction literals and historical context are retained above; the executable harness uses the existing is-even fixture with the same go run exit-code defect, exercising Shape and the real step analyzer rather than relying on an invented new named implementation test. Add a regression using the supplied three-file Go Greet example during implementation as specified below.
- Addressed: request Constraints by preserving existing budgets and configuration, using only local scripted providers. No product scope is omitted or superseded; historical timings and request counts are diagnostic context, not required future performance values.

## Tests
- A final step-code defect is returned to the fake shaper and corrected bytes are rechecked before publication: add a Shape regression with the three-file Go Greet empty-name example and features/steps/empty-name.js, alongside the protected correction journey using the existing fixture (regression)
- Repeated defects exhaust the existing correction allowance and end intent/not_ready naming the remaining exit_code finding without publishing: protected exhaustion journey and focused existing-loop regression (regression)
- Existing readiness findings, CLI options and timeouts remain unchanged: retain submission retry and readiness tests and run configured checks (existing)

## Feasibility
- Inspected the local verification dependency's StepItems: exit_code defects explicitly identify go run and instruct go build; no external protocol changes or provider assumptions are required. Local dependency inspection probe exited 0.
- evidence: {"id":"step-repair-observer","kind":"observer","inputs":["features/steps/k_stepcheck_repair_steps.js","features/steps/k_stepcheck_repair_test.go.txt"],"observers":["features/steps/k_stepcheck_repair_steps.js","features/steps/k_stepcheck_repair_test.go.txt"],"checks":[{"name":"good","command":"node features/steps/k_stepcheck_repair_steps.js good","exit":0,"output":"accepted\n"},{"name":"bad","command":"node features/steps/k_stepcheck_repair_steps.js bad","exit":1,"output":"rejected\n"}]}
