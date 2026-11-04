# contract

What Kogen relies on from kogen-shaping, as tests. Kogen is the only consumer: it pins an exact
version of this module and assembles the `kogen` program. The suite is the provider's side of the
contract: an interface change that would break Kogen fails here first, in kogen-shaping's own
`go test ./...`, without running any consumer suite.

## What it pins

Exactly what Kogen uses of kogen-shaping, nothing else: the exported API it calls, the behaviours it
relies on and the formats it reads. The evidence is Kogen at 9bf6073:

- `cli/engine.go` calls `shape.Shape(ctx, s, shape.ShapeInput{Project, Slug, Request, RequestPath,
  Tools, InstallRoot, Client, External})`, `shape.Refine(ctx, s, shape.RefineInput{Project, Slug,
  Hash, Request, RequestPath, Tools, InstallRoot, Client, External})` and `shape.Approve(ctx, s,
  shape.ApproveInput{Project, Slug, Hash, By, Tools})`; `cli/facts.go` calls `shape.Preflight(ctx,
  s, shape.PreflightInput{Project, Slug, JSON, Checkout})`. `s` is a `*sys.Session`, `ctx` a
  `context.Context`. Each returns `(exit code int, error)`, and Kogen maps the exit code to its own
  process exit. `Tools` is a `verify.Tools`, `Client` a `*provider.Client`, `External` Kogen's
  optional external executor (`shape.External`).
- `cli/response_budget_test.go` runs `TestShapeBudgetRolesShape` and requires four of its subtests to
  pass: "shape sends every request under thirty minutes and records it once", "a stalled shape
  expires unavailable naming thirty minutes, is not retried and records once", "refine sends every
  request under thirty minutes and records it once" and "a stalled refine expires unavailable naming
  thirty minutes and records once". `cli/assembly_test.go` only lists the module path.

## Inventory

| Pinned item | Asserted by |
| --- | --- |
| `shape.Shape` has the type `func(context.Context, *sys.Session, shape.ShapeInput) (int, error)` | `api_test.go`: compile-time pin |
| `shape.Refine` has the type `func(context.Context, *sys.Session, shape.RefineInput) (int, error)` | `api_test.go`: compile-time pin |
| `shape.Approve` has the type `func(context.Context, *sys.Session, shape.ApproveInput) (int, error)` | `api_test.go`: compile-time pin |
| `shape.Preflight` has the type `func(context.Context, *sys.Session, shape.PreflightInput) (int, error)` | `api_test.go`: compile-time pin |
| `shape.ShapeInput`: `Project`, `Slug`, `Request`, `RequestPath`, `InstallRoot` (strings), `Tools` (`verify.Tools`), `Client` (`*provider.Client`), `External` (`shape.External`) | `api_test.go`: `inputs` (compile time), `TestInputFieldsKeepTheirMeaning` |
| `shape.RefineInput`: the same fields plus `Hash` (string) | `api_test.go`: `inputs`, `TestInputFieldsKeepTheirMeaning` |
| `shape.ApproveInput`: `Project`, `Slug`, `Hash`, `By` (strings), `Tools` (`verify.Tools`) | `api_test.go`: `inputs`, `TestInputFieldsKeepTheirMeaning` |
| `shape.PreflightInput`: `Project`, `Slug` (strings), `JSON` (bool), `Checkout` (`any`) | `api_test.go`: `inputs`, `TestInputFieldsKeepTheirMeaning` |
| `shape.External` and `func(project.Role) (executor.Executor, error)` are assignable to each other | `api_test.go`: compile-time pin |
| `TestShapeBudgetRolesShape`, copied from `shape/response_budget_test.go`, through `shape.Shape` and `shape.Refine` with no `ResponseBudget` configured | `budget_test.go`: the four subtests below |
| "shape sends every request under thirty minutes and records it once": every request carries a deadline 29 to 30 minutes away, `ResponseBudget` stays 0, one `response_budget` event (`budget` "30 minutes", `budget_ms` 1800000) in `.kogen/local/shaping/is-even/events.ndjson` | `TestShapeBudgetRolesShape` |
| "a stalled shape expires unavailable naming thirty minutes, is not retried and records once": exit 4, `*sys.Error` `shape`/`provider_failed` with `Exit` 4, the line `provider/unavailable: ChatGPT response wait exceeded 30 minutes` and not `30m0s`, one shaper request, one budget event; a rerun records a second event | `TestShapeBudgetRolesShape` |
| "refine sends every request under thirty minutes and records it once": the same deadlines, `ResponseBudget` 0, one budget event in the refinement evidence's `events.ndjson` | `TestShapeBudgetRolesShape` |
| "a stalled refine expires unavailable naming thirty minutes and records once": exit 4, `*sys.Error` `provider`/`unavailable` with `Exit` 4, the same line, one shaper request, one budget event | `TestShapeBudgetRolesShape` |
| The exit code of each refusal equals the returned error's `Exit` (4) | the two stalled subtests of `TestShapeBudgetRolesShape` |
| `shape.Approve` without a hash is the review card (exit 5, nil error); with the full hash and `By` it approves (exit 0, `approved is-even <8 hex>` and the `Next:` line, `.kogen/local/approvals/is-even.json` with `slug`, `sha256`, `approver`, `base`) | `approve_test.go`: `TestApproveExitCodes` |
| `shape.Approve` on a directory that is not a Git repository: exit 3, `*sys.Error` `environment`/`not_a_git_repo` with `Exit` 3 | `approve_test.go`: `TestApproveExitCodes` |
| `shape.Preflight` with `JSON` after an approval: exit 0 and one JSON line (`schema_version`, `kind`, the given `checkout`, five passed checks, no blockers, the limitation) | `preflight_test.go`: `TestPreflightJSON` |
| `shape.Preflight` with `JSON` on a repository without `.kogen/project.yaml`: exit 3, `"data":null` and the `error` of `project.LoadConfig` | `preflight_test.go`: `TestPreflightJSON` |

## How to run it

```sh
go test -race -count=1 ./contract/...
```

The package is a normal test package, so the configured `go-test` check
(`GOFLAGS=-trimpath go test ./...`) and the landing gate's `go test -race ./...` run it too. It runs
in a few seconds (at most 60 seconds are allowed for its package run time). It needs Node, and
Cucumber for Shape's step check: `KOGEN_NODE` (else `node` from `PATH`) and `KOGEN_CUCUMBER` (else
`testdata/tools/cucumber/node_modules`, after `npm ci --prefix testdata/tools/cucumber`).

## Isolation

- The package is made only of `_test.go` files; it uses the exported API of `shape` and packages of
  the Kogen dependency modules `go.mod` already requires. It never reads or imports Kogen or any
  other consumer repository.
- There is no network: the provider is an in-process `http.RoundTripper` set as the client's `HTTP`
  that answers on the Responses wire. The suite imports neither `net` nor `net/http/httptest`,
  opens no listener and starts no server. `KOGEN_PROVIDER_URL` is a loopback origin that is never
  dialed.
- Every file it writes is under `t.TempDir()`. The fixtures are synthetic: a Git repository with a
  local identity, a replayed Cucumber report and a signed-in home that holds fixture credentials.
- Tests run in parallel and pass deterministically. None raises a timeout, skips, or waits on the
  wall clock: a stalled provider expires by advancing the client's clock.

## Out of scope

`shape.Init` (moving to kogen-setup, absent from kogen-shaping), shape workers and the
`shape_batch` tool (removed from shaping; `shape_batch` answers "unknown tool"), and the "shape
workers send under the same thirty minutes within the one record" subtest. The suite covers only
what shaping provides today.

## Changing the interface

An intended interface change edits this contract together with the paired Kogen items (its calls and
tests listed above). A change to the exported API, the exit codes, the budget behaviour or the file
formats listed in the inventory fails this suite first; do not relax the suite to land such a change
without the paired Kogen change.
