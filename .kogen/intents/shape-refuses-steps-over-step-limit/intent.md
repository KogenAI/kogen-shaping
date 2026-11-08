# Refuse acceptance steps that run whole suites

Slug: shape-refuses-steps-over-step-limit
Model: not applicable — the step self-check is one static judgement of a candidate package with no workflow state
Acceptance:
- .kogen/intents/shape-refuses-steps-over-step-limit/acceptance.feature
- features/steps/shape-refuses-steps-over-step-limit__steps.js
- features/steps/shape-refuses-steps-over-step-limit__harness.js

Shape approves acceptance steps that cannot finish within the 120 s per-step approval limit. Readiness runs on the unchanged tree, where a step that runs a whole test suite fails fast, so Shape never sees its runtime. Two real cases: the step `I run two uncached full Go suites concurrently` of usage-limit-packaging-isolation-abort-under-load ran 114 s, acceptance was stopped at 120 s and the Build was recorded intent/not_ready; and k-shape-harness-clean-env-lifecycle ended environment/step_timeout at 120 s.

The existing step self-check (`stepSelfcheck` in `shape/stepcheck.go`) already loads the candidate's own step files with Kogen's Node and Cucumber, without running any body, and refuses before readiness. It gains one more judgement: a step that obviously runs a whole test suite is refused, with a message that names the step and says to narrow it or split it below the step limit. Steps that run one named test or one package stay accepted.

What the check reads: `StepDef.Body`, the source of each step definition, of every definition that the feature's steps use (`Used`) and that is in one of the Intent's own step files. It never runs a body. A helper file or function that a body calls is not read; only commands written in the body are judged.

What counts as a whole suite. A command is written either as one shell string (`'go test ./...'`, also after `&&`, `||`, `;` or `|`, and after leading `NAME=value` assignments) or as a program name followed by an argument list (`'go', ['test', './...']`). It is a whole suite when it is:
- `go test` whose arguments include `./...` and no `-run`, `-run=...` or `-test.run...`;
- `mix test` whose arguments are all flags starting with `-`, none of them `--only`, `--failed` or `--stale` (a path or a value narrows it);
- `npm test`, `pnpm test`, `yarn test` or `npm|pnpm|yarn run test` whose remaining arguments are all flags starting with `-` and none is `--` (anything after `--` is forwarded to narrow the run).
Several whole suites in one step, such as two concurrent `go test ./...`, are the same defect: the step is refused once. `go test ./shape`, `go test -run '^TestOne$' ./shape`, `mix test test/one_test.exs` and `npm test -- --grep one` are narrow and accepted.

The refusal is one line per refused step definition, in the step self-check feedback after the existing headline `Step self-check failed; nothing was published. Fix the step files and resubmit.`, in the same `- <check> <target>: <detail>` form as the other non-binding lines:

`- step_limit <definition pattern>: runs a whole test suite with <runner> and no narrowing; narrow it to one named test or package, or split the step, so it finishes below the <N> s step limit`

`<definition pattern>` is `StepDef.Pattern`, which for a plain string expression is the step sentence. `<runner>` is `go test`, `mix test` or `npm test` (`pnpm test`, `yarn test` for those programs): the runner of the first whole-suite command in the body. `<N>` is `verify.StepTimeout` in whole seconds, today 120. The same item, with check `step_limit`, the definition pattern as target and its file, is recorded in the `step_selfcheck` event. Existing headline, binding, ownership, load and observer items, warnings, readiness, approval and publication are unchanged. No prompt, policy, tool definition, CLI flag or golden file changes.

The limit is the constant approval already uses, `verify.StepTimeout` (kogen-verification, already used by `shape/unpassable.go` and `shape/static.go`). The new code reads it and formats the number from it; it declares no 120, no 120-second duration and no spelled-out `120 s` of its own.

## Decisions
- The refusal is a static judgement in the existing step self-check, over the source of the used step definitions of the Intent's own step files, before readiness; no step body is run. (decision: D1, round 1) (constraint: reuse verify.RunStepScript's StepDef.Body; add no second step loader) (evidence: shape/stepcheck.go; shape/unpassable.go) (source: request)
- A step is a whole suite when it runs `go test ./...` without `-run`, `mix test` without a path or narrowing flag, or `npm|pnpm|yarn test` without `--` forwarding, written as a shell string or as a program with an argument list; several concurrent whole suites in one step are the same single refusal. (decision: D2, round 1) (constraint: `go test ./shape`, `go test -run ... ./shape`, `mix test <path>` and `npm test -- <args>` are accepted) (evidence: shape/static.go) (assumption: other runners such as pytest or cargo test are not judged, because the request names go test ./..., mix test and npm test; confirm that is enough) (source: request)
- The refusal line is `- step_limit <definition pattern>: runs a whole test suite with <runner> and no narrowing; narrow it to one named test or package, or split the step, so it finishes below the <N> s step limit`, which names the step and the limit and says to narrow or split it. (decision: D3, round 1) (constraint: <N> is formatted from verify.StepTimeout) (evidence: shape/stepcheck.go) (source: request)
- The limit is `verify.StepTimeout`, the constant approval uses, with no second copy in the shape package. (decision: D4, round 1) (constraint: no 120, no 120-second duration and no spelled-out `120 s` in a non-test Go file of shape) (evidence: shape/unpassable.go; shape/static.go; consumes/kogen-verification/steps_test.go) (source: request)
- Only commands written in a step definition's own source are judged; a body that calls a helper which runs a suite is not detected. (decision: D5, round 1) (assumption: "obviously run whole test suites" in the request means commands visible in the step body; confirm helper-hidden suites may pass) (source: request)

## Source comparison
- Addressed: request Problem, Evidence and Expected. The whole-suite refusal naming the step and the 120 s limit, with the narrow-step acceptance, is the rules above and the executable scenarios. The two historical cases (usage-limit-packaging-isolation-abort-under-load, k-shape-harness-clean-env-lifecycle) are supplied motivation, not local fixtures; the step sentence `I run two uncached full Go suites concurrently` is kept verbatim in acceptance.
- Addressed: acceptance check 1 (whole suite refused naming step and limit): the refusal scenarios for go test, mix test, npm test, the argument-list form and the concurrent step.
- Addressed: acceptance check 2 (single named test or package passes): the accepted scenarios, `go test -run ... ./shape`, `go test ./shape`, `mix test <file>` and `npm test -- <args>`.
- Addressed: acceptance check 3 (same constant, no second copy): the scenario that reads the limit from `verify.StepTimeout` in the harness and scans the non-test shape sources.
- Addressed: acceptance check 4 (existing step-check tests and the kogen-shaping suite pass): `go test ./...` and the configured checks stay in force; the existing step-check tests in `shape/stepcheck_test.go` are not edited.
- Omitted: refusing by measured runtime, judging helper files, and runners beyond go, mix and the npm family: the request asks for an obvious static refusal.
- Superseded: none. The `step_selfcheck` event gains `step_limit` items only for refused steps.

## Tests
- A step that runs a whole suite (go test ./..., mix test, npm test, shell string or argument list, one or several concurrent) is refused naming the step and the limit: refusal scenarios through the scripted shaper, and focused unit tests of the whole-suite judgement for each runner in both forms (scenario)
- Narrow steps (one named Go test, one Go package, one Elixir file, npm test with forwarded arguments) are accepted: accepted scenarios and unit tests of each narrowing rule (unit)
- A narrow step beside a whole-suite step is not named: the mixed scenario (scenario)
- The limit is verify.StepTimeout with no second copy: the approval-limit scenario and a focused test that the feedback number follows verify.StepTimeout (unit)
- Existing step-check behaviour is unchanged: the existing step self-check tests and `go test ./...` (existing)

## Feasibility
- evidence: {"id":"step-limit-observer","kind":"observer","inputs":["features/steps/shape-refuses-steps-over-step-limit__steps.js","features/steps/shape-refuses-steps-over-step-limit__harness.js"],"observers":["features/steps/shape-refuses-steps-over-step-limit__steps.js"],"checks":[{"name":"good","command":"node features/steps/shape-refuses-steps-over-step-limit__steps.js good","exit":0,"output":"accepted\n"},{"name":"bad","command":"node features/steps/shape-refuses-steps-over-step-limit__steps.js bad","exit":1,"output":"rejected\n"}]}
