# Check shaped files with configured static checks

Slug: k-shape-artifact-static-checks
Model: not applicable — each check run is a function of the exact candidate bytes, the configuration and the environment; the only retained state is a result keyed by HEAD, candidate tree and package hash, reused only for that identical identity, with no retry, ordering or interacting rule to model
Acceptance:
- .kogen/intents/k-shape-artifact-static-checks/acceptance.feature
- features/steps/k-shape-artifact-static-checks__steps.js
- features/steps/k-shape-artifact-static-checks__harness.js

Request (kogen-shaping, "static checks of shaped artifacts"). Problem: a generated acceptance helper can load and exercise the requested behavior yet fail a repository check that applies to the helper itself. Once approved, that helper is protected: Build cannot fix its formatting or static errors without changing the approved contract. Running configured fixers does not establish that configured checks pass, especially when the repository has no fixer. Need: before a shaped package can be approved, establish that its generated or changed files satisfy the repository's configured static checks applicable to those paths, including format, lint and vet; check the final shaped bytes in a fresh readiness copy with the repository's setup, toolchain and configuration; use the configured commands and their actual scope, never a weaker per-file approximation. A defect in a shaped file returns repairable feedback naming the configured check, the file and location when available, the exit status and bounded redacted output; unchanged acknowledgment cannot waive it; repaired bytes are rechecked and the approved hash identifies the checked bytes. Existing fixer behavior is preserved and unrelated files stay protected from fixer changes. An unrelated pre-existing failure or an unavailable checker is reported honestly, without declaring the shaped files checked and without requiring unrelated edits. No check is weakened, no useful assertion deleted, no timeout expanded and no whole test suite run merely to check shaped artifacts. A new behavior's outcome assertion may stay red on unchanged production code after its acceptance artifacts pass their static checks. This is one artifact-validity requirement, independent of contract review.

Current state of the code: `checkSubmission` in shape/shape.go writes the accumulated package, runs the configured fixers in the shaping checkout (`fixAttempt`, which undoes fixer changes to every other file), then the decision, literal and step self-checks and readiness (`certify`, kogen-verification's clean materialized copy: setup, Feasibility, model, acceptance); the contract review follows. No configured check (`commands.checks`) runs anywhere in Shape or `kogen intent approve`, so a helper that fails `gofmt` or `go vet` is published and approvable whenever its acceptance loads. Submission validation and step ownership (`scopedStepPath` in shape/stepownership.go) accept only `<steps_dir>/<slug>__<suffix>.js`, so a helper file such as `features/steps/static-proof__driver/main.go` cannot be submitted at all.

After this change, `kogen intent shape` (packages `shape` and `cmd/kogen-shape`, native and claude-code shapers alike) runs a static gate on every files submission that passes the step self-check, before readiness, and `kogen intent approve` runs it (or reuses Shape's identical result) before showing the card or approving. The gate runs the configured static checks unchanged in a fresh copy of the exact candidate, attributes failures to the shaped files only when they are theirs, refuses attributable defects with repairable feedback, and reports pre-existing failures and unavailable checkers as not established. Shape and Approve gain no option, flag, setting or environment variable for it. A helper file inside a slug-owned directory under the steps folder becomes a valid acceptance artifact.

Synthetic fixture of the acceptance, verbatim from the request (request: make Count return the slice length). go.mod:

```go
module example.test/staticshape

go 1.27
```

count/count.go:

```go
package count

func Count(items []string) int { return 0 }
```

.kogen/project.yaml (the runner supplies KOGEN_NODE and KOGEN_CUCUMBER as for ordinary acceptance; case 4 uses `fix: ['gofmt -w features/steps']`):

```yaml
version: 1
commands:
  setup: []
  fix: []
  checks:
    - name: layout
      run: 'test -z "$(gofmt -l count features/steps)"'
    - name: analysis
      run: 'go vet ./...'
  acceptance:
    run: 'NODE_PATH="$KOGEN_CUCUMBER" "$KOGEN_NODE" "$KOGEN_CUCUMBER/@cucumber/cucumber/bin/cucumber.js" --strict --require "features/steps/*.js" --format "message:$KOGEN_ACCEPTANCE_REPORT" "$KOGEN_ACCEPTANCE_FEATURES"'
    steps_dir: features/steps
protect: []
```

The shaped package lists the helper as well as the step module. `features/steps/static-proof__driver/main.go`, intentionally unformatted:

```go
package main
import("fmt";"example.test/staticshape/count")
func main(){fmt.Println(count.Count([]string{"a","b"}))}
```

`acceptance.feature`:

```gherkin
Feature: Count items
  Scenario: Observe the count
    When static-proof: I count two items
    Then static-proof: the count is 2
```

`features/steps/static-proof__steps.js`:

```js
const assert = require('node:assert/strict');
const { execFileSync } = require('node:child_process');
const { When, Then } = require('@cucumber/cucumber');
When('static-proof: I count two items', function () {
  this.value = execFileSync('go', ['run', './features/steps/static-proof__driver'], {
    cwd: process.cwd(), encoding: 'utf8'
  }).trim();
});
Then('static-proof: the count is {int}', function (expected) {
  assert.equal(this.value, String(expected));
});
```

The helper replaced by its gofmt output (case 2):

```go
package main

import (
	"fmt"

	"example.test/staticshape/count"
)

func main() {
	fmt.Println(count.Count([]string{"a", "b"}))
}
```

The uncalled function added to that formatted helper, which stays gofmt-clean (case 3), and repaired by changing `%d` to `%s`:

```go
func diagnostic() {
	fmt.Printf("%d", "not-a-number")
}
```

The independently unformatted pre-existing `count/unrelated.go` of case 5 contains `package count;var Existing=1`.

Out of scope and unchanged: Refine's conversation (a refined package is checked when it is approved); the Build; kogen-verification, kogen-coordinator and the other pinned modules (go.mod, go.sum); every prompt under shape/prompts, `ShaperInstructions`, the tool definitions and the policy and force goldens (shape/testdata/*golden*.json), byte for byte; the configured fixers and `fixAttempt`; the decision, literal, question and contract checks and readiness; this repository's `.kogen/project.yaml`; and the files of earlier Intents.

## Decisions
- Placement in Shape: in `checkSubmission`, after the step self-check returns no feedback and before `kogen: checking readiness`, on every files submission that reaches that point (including corrected and acknowledgment submissions). A refused result calls `sh.refused("static_checks", <problem lines>)` and returns an invalid outcome with the feedback text, so readiness, the contract review and acknowledgment never run for it and the existing `Unchanged: this package is identical to the one refused at attempt <n>; repair it before submitting again.` suffix applies to an identical resubmission. Any other result continues unchanged: the shaper receives exactly what readiness returns today and gets no extra turn. (source: request)
- Shaped files: every regular file under `.kogen/intents/<slug>/` and every `Acceptance:` path of the candidate, as written after the fixers (in Approve: of the package on disk), each once, sorted by path. (source: request)
- Fresh copy: the candidate tree `repo.Tree(repo.Head(), nil)` of the shaping checkout after writing and fixing (in Approve: `ready.CandidateTree`), materialized with `repo.Materialize` into a new directory `sys.TmpPath("kogen-static")` outside the checkout and removed afterwards; the configured `commands.setup` then runs there through `(*verify.Readiness).Setup(ctx, s, dir, false)` (in Approve: a Readiness value without `Attempt`), whose failure ends the run exactly as readiness's setup failure does. The SHA-256 of each shaped file is read in the copy before any check runs. No configured check ever runs in the shaping checkout. (source: request)
- Selection: every configured check (`cfg.Checks`) in configured order, except a test-suite check, which is recorded `not_run` and never executed. A check is a test-suite check when its `run` matches the Go regular expression `(?:^|[\s;&|(\x60])(?:(?:go|cargo|dotnet|mix|deno|bun)\s+test|(?:npm|pnpm|yarn)\s+(?:run\s+)?test|pytest|jest|vitest|mocha|rspec)(?:$|[\s;&|)\x60])`; so `GOFLAGS=-trimpath go test ./...` is not run and `test -z "$(gofmt -l .)"`, `go build ./...` and `go vet ./...` are. Names never matter. Without any selected check no copy is made, the outcome is `none` and the progress line is `kogen: no configured static check applies to the shaped files`. (source: request)
- Execution: each selected check runs its configured `run` unchanged through `s.Run` (`/bin/sh -c`) in the copy's root, with the session's environment and `sys.Options{Timeout: verify.StepTimeout, KillGroupOnExit: true}`. Its status, in this order: `unavailable` when it timed out (reason `timed out after 120 s`), when a line of its output matches `(?m)^(?:\S*/)?(?:sh|bash|dash|zsh|ksh): (?:line [0-9]+: |[0-9]+: )?([^\s:]+): (?:command )?not found\s*$` (reason `<captured program> not found`, whatever the exit status, so `/bin/sh: gofmt: command not found` with exit 0 is unavailable) or when it exited 126 or 127 (reason `exit <status>`); else `passed` on exit 0. (source: request)
- Attribution of a check that exited nonzero and is not unavailable: Kogen runs it once more as the command `set -x` followed by a newline and the configured `run` (its status is ignored; its output is the trace, used only to name paths), then removes every shaped file from the copy (empty directories stay), runs the configured `run` again (the baseline run), and writes every shaped file back with its bytes and mode, verifying their SHA-256 before the next check. The check is `attributable` when the baseline run passes (exit 0, not unavailable), or when the output names a shaped file that the baseline run's output does not name; otherwise it is `preexisting`. (source: request)
- Naming: a repository path p is named by a text at an occurrence of p whose preceding byte (if any) is not a letter, digit, `_`, `-` or `.` and whose following byte (if any) is not a letter, digit, `_`, `-` or `/`, nor a `.` followed by a letter or digit. A `:<line>` or `:<line>:<column>` (digits) directly after an occurrence is its location, and an entry is `p:<line>[:<column>]` from p's first located occurrence in the output, else in the trace, else plain p. An attributable check's entries are the shaped files named by its output or, when the output names none, by its trace; a pre-existing check's entries are the files of the candidate tree that are not shaped files and are named by its output or trace; entries are sorted by path. (source: request)
- Outcome: `refused` when any check is attributable; else `unknown` when any check is preexisting or unavailable; else `checked` (or `none` without a selected check). Every selected check runs, so one refusal names every attributable check. Shape keeps the last result of the run with its HEAD and candidate tree; a later submission with the same HEAD and tree reuses it without a copy or command, prints `kogen: static checks: the candidate is unchanged since submission <n>; their result stands` and refuses again when it was refused. Changed bytes always get a fresh run. (source: request)
- Feedback of a refusal, joined by newlines: the headline `Static checks: the shaped files fail <n> configured static check(s); nothing was published. Each check ran unchanged on the final submitted bytes in a fresh copy of the candidate; repair the named files so that every check passes. Resubmitting this package unchanged is refused again.` with n the number of attributable checks, then for each attributable check in configured order the problem line `- check <name>: exit <status>; files: <entries joined by ", ", or (not named by the check)>; command: <run>` followed by one line `  | <line>` per line of its output passed through `verify.Diagnostic` and the redactor `verify.NewRedactor(<session environment>, <HOME>)`, or the single line `  | (no output)` when that is empty. The problem lines (without the headline) are what `submission_refused` records. (source: request)
- Progress lines (through `sh.progress` in Shape, `s.Progress` in Approve), first `kogen: running static checks on the shaped files in a fresh copy: <selected names joined by ", ">`, then one per configured check: `kogen: static check <name> passed`; `kogen: static check <name> failed (exit <status>) because of the shaped files: <entries or (not named by the check)>`; `kogen: static check <name> failed (exit <status>) without the shaped files too: pre-existing failure in <entries or (no path named)>; the shaped files are not established by this check`; `kogen: static check <name> unavailable: <reason>; the shaped files are not established by this check`; `kogen: static check <name> not run: it runs the test suite`. (source: request)
- Evidence: each gate run in Shape records `{"event":"static_checks","attempt":<submission number>,"outcome":...,"reused":<bool>,"dir":<absolute copy directory>,"head":...,"tree":<candidate tree id>,"sha256":<package hash, sh.packageHash()>,"files":[{"path","sha256"}],"checks":[{"name","run","status","exit","paths","reason","output","trace"}]}`, where `status` is passed, attributable, preexisting, unavailable or not_run, `exit` is omitted only for not_run, `paths` holds the entries (always an array), `reason` is set only for unavailable, and `output` and `trace` are bounded and redacted as in the feedback; a reused result is recorded with the original fields, the current attempt and `reused` true. Every fresh run also writes `.kogen/local/static-checks/<slug>.json`, `{"version":1}` plus the same outcome, dir, head, tree, sha256, files and checks. (source: request)
- Approve: in `approveLocked`, right after `rd.Current(s, slug, ready)` is confirmed, it reuses `.kogen/local/static-checks/<slug>.json` when its version is 1, its head equals `ready.Head`, its tree equals `ready.CandidateTree`, its sha256 equals the package hash `sha` and its outcome is checked, unknown or none (printing `kogen: static checks: reusing the result recorded for this package (<outcome>)`); otherwise it runs the gate itself on `ready.CandidateTree` with the progress lines above and writes the record. A refused result fails with `sys.Fail("intent", "static_checks", "the Intent's files fail configured static checks", 1)` whose Lines are the problem lines; no card is printed and no approval is written. Unknown or none continue to the card or the approval unchanged. (source: request)
- Helper files: `scopedStepPath` also accepts `<steps_dir>/<slug>__<suffix>/<rest>` with a nonempty suffix without `/` and a nonempty normal rest, so submission validation and step ownership accept `features/steps/static-proof__driver/main.go` for slug static-proof; such a file must still be submitted by the Intent itself, and only `.js` files directly under the steps folder are step modules, as today. Messages are unchanged. (source: request)
- Fixers are unchanged: they still run in the shaping checkout before the gate, and `fixAttempt` still undoes their changes to every file outside the attempt; the gate checks the bytes they produce, and the approval hash covers those same bytes. (source: request)
- The acceptance harness relies on these existing test helpers keeping their names and signatures: `newShpFx`; the `shpFx` fields `Root`, `Env`, `S`, `Stdout`, `Stderr`, `Prov` and `Client` and its methods `git`, `write`, `read` and `newSession`; `shpStep`, `shpSubmit`, `shpOutputs`, `shpToolNames`, `contractKind`, `decisionRequest`, `literalRequest`, `SubmitQuestionCheckTool` and the `shpProvider` method `allRequests`; `ShapeInput.SubmissionRetryLimit`; and `Approve` with `ApproveInput`. Tests are synthetic only, with no live provider, and use only owned child processes and temporary folders. (source: request)

## Source comparison
- Need "establish that generated or changed files satisfy configured static checks ... including format, lint and vet" and "check the final shaped bytes in the readiness fresh copy": the static gate (Decisions Placement, Fresh copy, Execution); scenarios "repair" (layout) and "vet" (analysis). kogen-verification's own readiness scratch is a pinned dependency whose `Gate` hook receives no directory, so the gate materializes the same candidate tree id that readiness checks into its own fresh copy, with setup; it never runs in the shaping checkout (scenario step "ran outside the checkout on the published bytes").
- "Use the configured commands and their actual scope; do not silently replace a whole-repository check with a weaker per-file approximation": checks run their `run` byte for byte; attribution reruns the same command (trace, baseline), never a narrower one; the event records `run` (scenario "unrelated", step "with its configured command").
- "naming the configured check, file/location when available, exit status and bounded redacted output": the problem line and `  | ` output lines; case 1 names `layout`, `features/steps/static-proof__driver/main.go`, exit 1 and the (empty) output; case 3 names `analysis` and `features/steps/static-proof__driver/main.go:14:14` with the vet message.
- Case 1 "refuse before approval ... Readiness's legitimate count assertion red cannot excuse this separate artifact defect. Repeat the unchanged submission and acknowledgment: it stays refused": the gate precedes readiness; case "repair" submissions 1 (whole package), 2 (unchanged) and 3 (acknowledgment-style, acceptance.feature only) are all refused by `static_checks`. The contract review is scripted to report no findings (the fake provider's default).
- Case 2 (replace only the helper with its gofmt output; both checks pass; Count still returns 0; red `'0' !== '2'` follows ordinary acknowledgment; no production change): "repair" submission 4 replaces only the helper, passes with outcome checked, receives readiness observations containing `'0' !== '2'`, and submission 5 acknowledges; count/count.go is unchanged and the package is approved.
- Case 3 (uncalled `diagnostic()` with `%d`, gofmt-clean, compiles, same runtime observation; `analysis` refuses pointing to the helper; `%s` repairs it): case "vet".
- Case 4 (`fix: ['gofmt -w features/steps']`, original unformatted helper; verify the fixed bytes, accept only their hash, keep production and unrelated files byte for byte; a change after checking cannot use the old check or hash): case "fixer" publishes the real gofmt output (which differs from the case 2 block in import order and layout), records the checked file hashes and package hash, keeps go.mod, count/count.go and .kogen/project.yaml unchanged; after the helper gains the `%d` function, approving the published hash is refused (hash mismatch) and approving the new hash is refused by `analysis` (Decision Approve).
- Case 5 (unrelated unformatted `count/unrelated.go`; report its path, no attribution, no alteration, no narrower command; checker absent reports unavailable): case "unrelated" records layout `preexisting` naming `count/unrelated.go` (its path comes from the trace `+ test -z count/unrelated.go`, because `test -z "$(...)"` prints nothing), outcome unknown, file unchanged; case "absent" removes every PATH directory holding `gofmt`, where `layout` exits 0 with `/bin/sh: gofmt: command not found` and is recorded unavailable.
- Case 6 (renamed checks and helper path, equivalent defects, same decisions; a valid first submission gets no repair turn; checks only in the readiness copy with evidence of directory and bytes): cases "renamed" (tidy, inspect, `features/steps/static-proof__counter/main.go` with both defects) and "valid" (published after exactly 2 submissions, no refusal); the event's `dir`, `tree`, `files` and `sha256`.
- "Distinguish an attributable shaped-file defect from an unrelated pre-existing failure or unavailable checker": Decisions Attribution and Outcome. A failure whose output is empty and which also fails without the shaped files is reported pre-existing even if a shaped file also contributes; it is never declared checked, and no exhaustive attribution is claimed.
- "Do not weaken checks, delete useful assertions, expand timeouts or run the whole test suite": checks are unchanged, bounded by the existing `verify.StepTimeout`, and test-suite commands are recorded not_run.
- "Recheck repaired bytes before approval; the approved hash must identify the checked bytes": fresh run on changed bytes; Approve reuses a result only for the identical package hash, HEAD and tree.
- The helper path `features/steps/static-proof__driver/main.go` must be submittable in "an otherwise valid Intent": Decision Helper files.

## Tests
- Test-suite selection by command (go test with an environment prefix, npm test, pytest not run; test -z with gofmt, go build, go vet run; names ignored): add TestStaticCheckSelection in shape/static_test.go (unit)
- Unavailable classification (exit 0 with the macOS and dash not-found lines, exit 127, exit 126, timeout) before passed and failed: add TestStaticCheckUnavailable in shape/static_test.go (unit)
- Path naming boundaries, absolute and ./ prefixes, locations and entry order: add TestStaticCheckNamedPaths in shape/static_test.go (unit)
- Problem lines, output lines, (no output), headline count and progress lines: add TestStaticCheckFeedback in shape/static_test.go (unit)
- Attribution with real commands in a materialized copy (baseline pass attributable; output-only attribution; pre-existing with path from the trace; shaped bytes restored; copy outside the checkout; record written): add TestStaticChecksAttribution in shape/static_test.go (integration)
- Helper files inside a slug-owned directory accepted by validation and ownership; other paths still refused: add TestScopedStepPathHelperDir in shape/stepownership_test.go (unit)
- Approve reuses a matching record, reruns and refuses a changed package with intent/static_checks and writes no approval: add TestApproveStaticChecks in shape/approve_test.go (integration)
- Shape refuses with gate static_checks before readiness, reuses an unchanged result and records the event: acceptance harness cases; existing Shape and Approve tests keep their behavioural assertions, updated only where they pin the exact progress lines or event list that now include the static gate (integration)
- The request's synthetic cases end to end through Shape and Approve with a scripted shaper and real Cucumber: acceptance.feature (scenario)
- Prompts, tool definitions and goldens stay byte-identical: the existing policy and force golden tests pass unchanged (existing)

## Feasibility
- Observed with Go 1.27.2 in a scratch copy of the fixture: `test -z "$(gofmt -l count features/steps)"` exits 1 with empty output for the unformatted helper and 0 for the case 2 block; `go vet ./...` prints `features/steps/static-proof__driver/main.go:14:14: fmt.Printf format %d has arg "not-a-number" of wrong type string` (exit 1) for case 3, `...:11:14` for the fixer output plus the function, and `features/steps/static-proof__counter/main.go:4:31` for the renamed both-defects helper; both checks pass once the helper is removed; gofmt of the original helper gives the bytes in the "fixer" scenario; with gofmt absent `/bin/sh -c` prints `/bin/sh: gofmt: command not found` and layout exits 0; `set -x` shows `+ test -z count/unrelated.go`; the real Cucumber run fails the scenario with `'0' !== '2'`.
- The harness compiles with `go test -c -vet=off -overlay` on the unchanged tree. There every case is refused at submission validation (the helper path), so every scenario is red at its first static-check assertion. With a scratch overlay that only widened `scopedStepPath`, cases valid, repair, fixer and absent ran in about 4-7 s each: published after acknowledgment with `'0' !== '2'` observations, the fixer's gofmt bytes published, the old hash refused as hash_mismatch and the changed package approved (today's gap).
- The judgments behind masked deeper assertions are qualified by the observer controls below. Good: one observation per case as the implemented gate would produce it. Bad: the unchanged-tree observations of every case, the fixer case without the gate, and single-field corruptions (wrong exit, file, command or missing output line, wrong gate, a narrowed command, an attributed or passed pre-existing failure, an unavailable checker recorded as passed, a copy inside the checkout, mismatched recorded bytes or hash, a missing or stale record, an approved changed package, extra refusals or turns).
- evidence: {"id":"static-checks-observer","kind":"observer","inputs":["features/steps/k-shape-artifact-static-checks__steps.js","features/steps/k-shape-artifact-static-checks__harness.js"],"observers":["features/steps/k-shape-artifact-static-checks__steps.js"],"checks":[{"name":"good","command":"node features/steps/k-shape-artifact-static-checks__steps.js good","exit":0,"output":"accepted\n"},{"name":"bad","command":"node features/steps/k-shape-artifact-static-checks__steps.js bad","exit":1,"output":"rejected\n"}]}
