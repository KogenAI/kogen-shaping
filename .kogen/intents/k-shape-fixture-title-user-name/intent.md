# Keep approve fixture titles free of the user name

Slug: k-shape-fixture-title-user-name
Model: not applicable — the change concerns literal test-fixture values (a Git user name and an Intent title) and has no state
Acceptance:
- .kogen/intents/k-shape-fixture-title-user-name/acceptance.feature
- features/steps/k-shape-fixture-title-user-name__steps.js

The approve fixtures in this module must pick a Git `user.name` and an Intent title so that the title never contains the user name as a whole word, ignoring case. kogen-shaping must not depend on the coordinator's title lint, and no kogen-shaping test may assert that lint's wording or its user-name rule. Coordinator `91d0a32` ("Refuse third-person owner commit titles") made Intent lint refuse a title that names the repository's effective `git config user.name`. The approve fixture (`newApproveFx` in `shape/approve_test.go`) configures `user.name` = `Fixture`. With the old title `# Fixture intent`, the fixture Intent stopped linting, cached readiness was not reused, and `TestApproveWithFullAndPrefixHash/6`, `/8` and `/64` failed with `cached readiness must not run acceptance again: 2 runs, had 1`.

The current tree already has this outcome: commit 6cefe25 ("Require kogen-loop by its real module path") changed `approveIntentMD` to `# Sample intent` and the card expectation in `TestApproveCardOutput` to `Intent: fixture — Sample intent`. This Intent records the outcome and guards it with acceptance. The Build needs no production change and must not change the coordinator pin, production code or the approver string. It changes a fixture only if acceptance shows that a fixture's title contains its own Git user name. In that case it changes the title, not what the test is about.

## Decisions
- The approve fixture keeps `user.name` = `Fixture`, `user.email` = `fixture@example.invalid` and the approval-record approver `Fixture <fixture@example.invalid>`, and its Intent title stays `# Sample intent`. The title does not contain the user name. (source: request)
- The guard is the Intent's acceptance, which scans the fixtures and runs the reproducing Go test. No kogen-shaping Go test is added that encodes the user-name rule or the coordinator's lint wording. (source: request)

## Source comparison
- Addressed: the approve fixture title `# Fixture intent` (`approveIntentMD`, shape/approve_test.go:25) with `user.name` = `Fixture` (shape/approve_test.go:105) is now `# Sample intent` at HEAD c4d2c1b. Running `go test -count=1 -run '^TestApproveWithFullAndPrefixHash$' -v ./shape` against the pinned coordinator `v0.0.0-20261105022408-e7bc7ce9c169` passes `/64`, `/8` and `/6`. In a scratch copy, restoring `# Fixture intent` made all three fail at approve_test.go:339 with `cached readiness must not run acceptance again: 2 runs, had 1`, so the pinned coordinator contains the user-name rule and the test reproduces the reported failure. Covered by the scenario "The approve regression passes against the pinned coordinator".
- Superseded: the request's "against kogen-coordinator main (`03d7341` today) and against the pinned `c4a49a4fe84f`". The module now pins `v0.0.0-20261105022408-e7bc7ce9c169`, which is later than both. With a `go mod edit -replace` to either `v0.0.0-20261103214825-03d7341feab7` or `v0.0.0-20261102193107-c4a49a4fe84f`, `./shape` no longer builds (`git/commit.go:10:2: malformed module path "kogen-loop": missing dot in first path element`), because those versions import kogen-loop by its old path. Acceptance therefore runs the test against the pin and requires the pin to be no older than `v0.0.0-20261103214825-03d7341feab7`.
- Omitted from executable acceptance: `consumer-check.sh github.com/KogenAI/kogen-coordinator <coordinator-main-checkout> <mirrors> 'kogen-shaping|<fix-sha>|.|'`. That script is not in this repository. Its predicate for this Intent, `TestApproveWithFullAndPrefixHash` passing against the coordinator, is the go-test scenario.
- Addressed ("check the other fixtures in the module that build an Intent title"): the acceptance scan finds three files that both configure a Git user name and build an Intent title (`# <title>\n\nSlug:`). They are `consumes/kogen-verification/helpers_test.go` (`Consumer`, `# Fixture intent`), `features/steps/k-approve-binding-check-batched__harness.js` (`Fixture`, `# Sample ` + slug, with slugs `syn-NN` and `syn-new`) and `shape/approve_test.go` (`Fixture`, `# Sample intent`). None clashes. `contract/preflight_test.go` passes `user.name=Fixture` only via `-c` for one commit and builds no Intent title.
- Addressed ("No kogen-shaping test asserts coordinator lint wording or the user-name rule"): no Go file and no step file other than this Intent's own contains `names the Git user`, `owner-authored title`, `narrates owner or user approval`, `third person` or `CheckTitle`.
- Kept: the approver string `Fixture <fixture@example.invalid>` asserted by `TestApproveWithFullAndPrefixHash`, and the card title assertion in `TestApproveCardOutput`, which is about the card. Out of scope: the title tests and protected-paths tests explained by coordinator 8d24cc7 and 4826fb1, and any other approve test failure not caused by the title clash.

## Tests
- The approve fixture's Intent title does not name its Git user, so approval reuses cached readiness against the pinned coordinator: existing `TestApproveWithFullAndPrefixHash` with subtests `64`, `8` and `6`, run by acceptance (existing)
- Every module fixture that configures a Git user name builds an Intent title without that name as a whole word, ignoring case: acceptance scan of the module's Go files and step files (integration)
- No kogen-shaping Go file or step file asserts the coordinator's title-lint wording or user-name rule: acceptance search of the module's Go files and step files (integration)

## Feasibility
- none
