# Batch step-ownership loaders per step-file pair

Slug: k-approve-binding-check-batched
Model: not applicable — the step-ownership check is one stateless judgement of the current packages and step files per call; only how many loader processes it starts changes, not any state or verdict
Acceptance:
- .kogen/intents/k-approve-binding-check-batched/acceptance.feature
- features/steps/k-approve-binding-check-batched__steps.js
- features/steps/k-approve-binding-check-batched__harness.js

The step-binding ownership check (`bindingCollision` in `shape/stepownership.go`) gives the same verdict as today but starts far fewer Node + Cucumber loader processes. Its callers are `kogen intent approve` (`approveLocked` in `shape/approve.go`, for the preview and for the approval with a digest), `kogen intent preflight` (`Preflight` in `shape/preflight.go`, the "package" check) and Refine (`shape/refine.go`). All three get the batched check; none of them keeps the old per-text path.

What the code does now: for every other Intent, every pair of this Intent's step file `a` and the other Intent's step file `b` (`a != b`), and every step text of both Intents' scenarios, it calls `verify.RunStepScript` once with files `[a, b]` and that single text. So it starts one Node process per (other Intent × file pair × step text): 600 processes for 30 other Intents of one step file and 20 texts per pair.

What the product does instead: one loader process per (own step file, other step file) pair. That process loads `a` then `b` and matches every step text of both Intents, returning per-text matches. The per-text `RunStepScript` loop in `bindingCollision` is removed.

The rule, exactly:

1. Iteration and skips are unchanged. Other Intents come in the order of `filepath.Glob(root/.kogen/intents/*/intent.md)`. The Intent's own slug is skipped, and so is any Intent whose `project.CheckIntent` fails. If this Intent's own `CheckIntent` fails, the check passes, as now. Own files come in `intent.StepFiles()` order, other files in `other.StepFiles()` order, and a pair with `a == b` is skipped.
2. The texts of a pair are every step text of this Intent's scenarios followed by every step text of the other Intent's scenarios (`project.Scenario.Steps[].Text`). Duplicates may be removed. A pair with no texts starts no loader and does not collide.
3. For each remaining pair, start exactly one loader process, run sequentially.
   - Node is `tools.Node`, else the session's `KOGEN_NODE`. Cucumber is `tools.Cucumber`, else the session's `KOGEN_CUCUMBER`. The process runs with `NODE_PATH=<cucumber>`, the working directory `root` and the session environment.
   - It loads the step files exactly the way kogen-verification's `stepScript` does: `supportCodeLibraryBuilder.reset(process.cwd(), …)`, then `require(path.resolve(f))` for `a` and then `b`, each in its own try/catch so a throwing file is recorded and loading goes on, then `finalize()`.
   - For each text it reports which loaded definitions match it under Cucumber's own `matchesStepName`, with each definition's `uri` file. It never calls a step body or hook.
4. A pair collides when at least one of its texts is matched both by a definition whose file is `a` and by a definition whose file is `b`, both for that same text. A definition of `a` matching one text and a definition of `b` matching a different text is not a collision.
5. The first colliding pair in the order of rule 1 refuses with today's error, unchanged: `sys.Fail("intent", "step_ownership_collision", fmt.Sprintf("Intents %s and %s have overlapping step bindings in %s and %s", slug, otherSlug, a, b), 1)` with `Lines = []string{a, b}`. The rendered error is `intent/step_ownership_collision: Intents <slug> and <other> have overlapping step bindings in <a> and <b>`, followed by `  <a>` and `  <b>`.
6. Load failures and missing results:
   - A file that throws while loading contributes only the definitions it registered before the throw. A file that throws before any definition contributes none, so it causes no refusal.
   - When the loader produces no result (Node cannot run, or no parsable result line), the pair is treated as not colliding and the check goes on, as today's `!loaded` → `continue`.
   - A cancelled context returns the context's error, as today.

With no collisions, the cost is one loader per step-file pair, not pairs × texts.

kogen-verification is a pinned dependency and is not edited. `verify.RunStepScript` reports only an any-text `used` flag per definition, so it cannot give per-text matches. The Build adds the batched loader script and its runner in this repository's `shape` package, modeled on `stepScript` and `RunStepScript`: the same loading, the result on one marked stdout line, and the same no-result fallback. `sharedStepCollision`, Shape's own step self-check (`shape/stepcheck.go`) and readiness's step analysis are unchanged.

Out of scope: the launcher's two approve calls (preview for the digest, then approve) stay as they are, and the approve card, digest and approval flow do not change. No CLI flag, prompt, policy, tool definition or golden changes.

## Decisions
- The step-ownership refusal keeps today's exact text, `intent/step_ownership_collision` reason, exit 1 and its two listed files, the same first collision in Intent order, and the same no-refusal handling of files that fail to load and of loaders without a result. (source: request)
- Batching is one loader per (own step file, other step file) pair, which matches every step text of both Intents in that process and returns per-text matches. The request's alternative, one shared library for all pairs, is not used. (source: request)
- The batched check applies to every caller: approve (preview and approval with the digest), preflight and Refine. (source: request)
- The launcher keeps calling approve twice per Build (preview for the digest, then approve). Nothing in approve's two-call flow changes. (source: request)

## Source comparison
- Addressed: request Problem. The per-text `verify.RunStepScript` loop in `bindingCollision` is replaced by one loader per step-file pair (rules 1–4).
- Addressed: Need, first bullet. One loader per (own step file, other step file) pair with every text of both Intents and per-text matches (rule 3). The second option, a single library per approve with per-pair isolation, is superseded by the decision above.
- Addressed: Need, second bullet. Unchanged verdict: the same pairs and order, the same rule, the same first collision, refusal text and listed files, and the same handling of load failures and of a loader with no result (rules 1, 4, 5 and 6).
- Addressed: Need, third bullet. With no collisions, the cost grows with step-file pairs (rule 3), checked by the "pairs" scenario.
- Addressed: Need, fourth bullet. All callers: approve, preflight and Refine share `bindingCollision`. The scenarios exercise approve and preflight end to end, and the Go tests cover the shared function.
- Addressed: Delivery note. The two approve calls are kept, and nothing in this repository changes there.
- Addressed: Synthetic acceptance, common setup. Each case uses a scratch Git repository with Kogen's normal layout and a `KOGEN_NODE` wrapper that appends one line per invocation and runs the real Node with the same arguments.
  - The whole acceptance command must finish within Kogen's fixed 120-second readiness limit (`verify.StepTimeout`). So the wrapper allows each Preflight or Approve call 40 Node invocations: at most 30 loaders plus readiness's own two, with room to spare.
  - Past that budget the wrapper logs each further invocation and refuses it at once, and the harness cancels the call's context. A judge then reports that call as over budget, without a verdict.
- Addressed: Synthetic acceptance 1, the "pairs" scenario.
  - Intents `syn-01` … `syn-30` each have `features/steps/syn-NN__steps.js` with ten steps `syn-NN: step K` (K = 1..10) and a feature using all ten. `syn-new` has `features/steps/syn-new__steps.js` with ten steps `syn-new: step K`.
  - Approve runs as preview, then with the digest from the preview card, and `syn-new` is approved.
  - The preview's check starts at most 30 loaders: the preview's Node invocations minus those of the same `syn-new` previewed alone, not 30 × 20. Preflight's check, which runs no readiness, is also counted directly.
- Addressed: Synthetic acceptance 2. `syn-new__steps.js` gains `Given('syn-17: step 4', …)`, and the refusal must be exactly `intent/step_ownership_collision: Intents syn-new and syn-17 have overlapping step bindings in features/steps/syn-new__steps.js and features/steps/syn-17__steps.js`, listing those two files. The same blocker is required from preflight.
- Addressed: Synthetic acceptance 3. With `syn-03: step 9` and `syn-17: step 4` both copied, the refusal names `syn-03`.
- Addressed: Synthetic acceptance 4. `syn-09__steps.js` begins with `throw new Error('syn-09 cannot load');`. `syn-new` also copies `syn-09: step 2`, so a check that matched patterns without loading would wrongly refuse. Expected: approved, with no refusal from that file.
- Addressed: Synthetic acceptance 5. `syn-alpha` and `syn-beta` are written and committed in two `git worktree` checkouts of the same repository, each with its own scoped step file and distinct texts. Both branches are merged into the main checkout, and each is approved there, as on the release. Each one's check starts at most one loader for its single pair.
- Omitted: Held-out acceptance. The Lab record, pack and timings (for example 16,360 s → 1,093 s pooled, and `kogen-shaping f04edce`) are evidence, not executable here. Re-timing approve on a kept Studio tree with about 100 Intents is an external measurement of trees and hosts that this repository does not have. No wall-clock assertion is added; the loader counts are the executable proxy.
- Omitted: Arm C (a single approve call). The request records it as not adopted.

## Tests
- One loader per step-file pair, with the same verdict and first collision: add a Go integration test to `shape/stepownership_test.go` that builds a temporary repository with the real Node (`KOGEN_NODE`) behind an invocation-counting wrapper and the Cucumber from `testdata/tools/cucumber`, and asserts one loader per pair and no error without collisions, and with collisions in two other Intents the exact refusal `sys.Error` (class, reason, detail, exit 1, `Lines`) of the first in Intent order (integration)
- A pair collides only when one and the same text matches a definition in each file: in the same test, an own definition matching only an own text and another file's definition matching only that Intent's text cause no refusal (integration)
- Files that fail to load and loaders without a result keep today's handling: in the same test, a step file that throws before defining anything causes no refusal, and a `KOGEN_NODE` that exits 1 without output yields no refusal and no error (integration)
- Approve preview, approval with the digest and preflight use the batched check with unchanged refusal text, first collision, load-failure handling and worktree packages: the five scenarios of acceptance.feature through the real `Approve` and `Preflight` entry points (scenario)
- Shared step files and preflight's ownership blocker stay unchanged: `TestSharedStepApprovalAndPreflight` (existing)
- Refine uses the same check: it calls the same `bindingCollision`, covered by the integration test above, and the existing Refine tests keep passing (existing)

## Feasibility
- First, a scratch Go overlay of the harness without a budget ran on the unchanged tree with the real Node 24.21.0 and the Kogen release's Cucumber. It established the release's counts and the verdicts that must be kept:
  - "pairs": preflight started 600 Node invocations; the preview started 602, against 2 for `syn-new` alone; the approval with the digest started 600 and approved.
  - "one-collision": 334 invocations, then exactly `intent/step_ownership_collision: Intents syn-new and syn-17 …` listing the two files, from approve and preflight.
  - "two-collisions": 59 invocations, naming syn-03.
  - "load-failure": the preview started 602 invocations and syn-new was approved.
  - "worktrees": 20 invocations per preflight, and both Intents were approved.
  - (probe: exit 0, about 5 min)
- The first readiness of this package timed out at the fixed 120-second acceptance limit. Hence the submitted harness's budget of 40 invocations per call with cancellation:
  - On the unchanged tree, all five cases finished in 40 s in total. Every "pairs", collision and "load-failure" call was cancelled at 41–42 invocations and is judged over budget. The worktree calls stayed within budget, with 20 and 22 invocations.
  - The warm overlay compile took 1.6–10 s, but up to about 3 min under host contention for the shared Go cache.
  - Passed through the submitted step definitions, only the worktree approval steps pass on these observations; all other steps fail with the over-budget or loader-count message. (probe: exit 0, 40 s)
- evidence: {"id":"approve-batch-observer","kind":"observer","inputs":["features/steps/k-approve-binding-check-batched__steps.js","features/steps/k-approve-binding-check-batched__harness.js"],"observers":["features/steps/k-approve-binding-check-batched__steps.js"],"checks":[{"name":"good","command":"node features/steps/k-approve-binding-check-batched__steps.js good","exit":0,"output":"accepted\n"},{"name":"bad","command":"node features/steps/k-approve-binding-check-batched__steps.js bad","exit":1,"output":"rejected\n"}]}
