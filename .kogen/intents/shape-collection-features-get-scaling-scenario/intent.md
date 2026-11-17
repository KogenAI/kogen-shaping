# Require a scaling scenario for collection requests

Slug: shape-collection-features-get-scaling-scenario
Model: not applicable — one fixed sentence added to a static prompt file and its recorded hashes; no state or interacting rules
Acceptance:
- .kogen/intents/shape-collection-features-get-scaling-scenario/acceptance.feature
- features/steps/shape-collection-features-get-scaling-scenario__steps.js

Shape never turns "the work must not grow per record" into an acceptance scenario for features that process collections, so a Build can ship per-record query loops that nothing in the Intent forbids. After this change the shaper instructions (`shape/prompts/shaper.md`) carry one rule: when the request describes an operation over a collection of records of unknown size, the acceptance package includes one scenario that runs the operation twice with different record counts and asserts that its work against its data store or external services stays within one stated upper bound for both counts, using the project's own test tooling; the scenario is skipped only with a `- none: <reason>` line in `## Tests` that says why the operation cannot grow. The rule is generic: framework, language and domain names appear only as marked examples. Requests that do not process a collection are unchanged. No gate, tool, setting or configuration is added, and `shape/prompts/refine.md` is not changed because it does not restate the scenario-writing rules.

The exact line added to `shape/prompts/shaper.md`, directly after the bullet that starts `- Read the project with read_file`, is given in the first acceptance scenario. The Build also updates the recorded prompt bytes: `shape/testdata/prompts_golden.json` (the `shaper` entry only) and `shape/testdata/shaper_policy_golden.json`, both rewritten with `go test ./... -update` and reviewed so that only the shaper prompt entries change, and adds a model-free Go regression test beside `shape/policy_golden_test.go` (in that file) that checks the prompt rendered by `shaperPrompt("shaper")` contains the collection rule and its skip rule.

## Decisions
- The rule is one bullet in the "How to shape" list of the shaper prompt, directly after the bullet about reading the project, with the exact wording of the first acceptance scenario; it names the two record counts, the same upper bound for both, the project's own test tooling, a controls.json wrong implementation that works once per record, the `- none: <reason>` skip and the no-collection exemption, and gives every framework-like term only as an example. (evidence: shape/prompts/shaper.md; shape/policy.go) (source: request)
- `shape/prompts/refine.md` stays unchanged: it does not repeat the scenario-writing rules, so the rule is not duplicated there. (decision: D2, round 1) (evidence: shape/prompts/refine.md) (source: request)
- The two goldens that hold the shaper prompt bytes, `shape/testdata/prompts_golden.json` and `shape/testdata/shaper_policy_golden.json`, are regenerated with the repository's `-update` flow; the other entries of `prompts_golden.json` keep their bytes and hashes. (decision: D3, round 1) (evidence: shape/policy_golden_test.go; shape/policy_test.go; shape/testdata/prompts_golden.json; README.md) (source: request)
- Acceptance renders the shaper prompt in Node the way `shaperPrompt` does (each `{{name}}` replaced by the text of `shape/prompts/<name>.md`) and compares its byte count and SHA-256 with the `shaper` entry of `prompts_golden.json`; the policy golden is covered by the existing Go test `TestShaperPolicyGolden`. (decision: D4, round 1) (evidence: shape/policy.go; shape/policy_golden_test.go; shape/policy_test.go) (assumption: The Node rendering equals the Go rendering because `agent.Render` only substitutes `{{name}}` placeholders; a scratch run in this session showed the match, but it is not a Kogen probe record) (source: request)
- One wrong-implementation control is supplied: the shaper prompt without the rule. No variation per operation kind is added. (decision: D5, round 1) (evidence: shape/prompts/shaper.md) (assumption: The request states that one control is enough; the user confirms that a prompt without the rule is the only control needed) (source: request)
- One wrong-implementation control is supplied: the shaper prompt without the rule. No variation per operation kind is added. (decision: D1, round 1) (evidence: shape/prompts/shaper.md) (assumption: The request states that one control is enough; the user confirms that a prompt without the rule is the only control needed.) (source: request)

## Source comparison
- Addressed: the shaper instructions require a two-record-count scaling scenario for collection operations, with the project's own test tooling and a stated bound; acceptance scenario 1 checks the exact line.
- Addressed: the skip is allowed only with an explicit `- none: <reason>` line in `## Tests` saying why the operation cannot grow; the same line carries it.
- Addressed: the instruction is generic, with examples marked as examples, and requests that do not process a collection get no such scenario; the same line says so.
- Addressed: a wrong implementation that does the work per record must fail the scenario, tied to the existing controls.json requirement; the line says the wrong implementation does the work once per record.
- Addressed: the golden is updated next to `shape/policy_golden_test.go` and a model-free regression test checks the rendered prompt; acceptance scenario 1 checks the prompts golden, and `## Tests` lists the Go test.
- Omitted: `shape/prompts/refine.md` (D2), a new gate, tool, configuration or per-operation-kind variations; none is requested.
- Omitted: the lab evidence paths named in the request (grade logs and analysis) are outside this repository and are not needed.

## Tests
- The rendered shaper prompt carries the collection scaling rule and its skip rule, and its recorded bytes match: acceptance scenario 1, whose controls.json wrong implementation removes the rule (scenario)
- The rendered shaper prompt contains the collection rule and its `- none: <reason>` skip rule: a new model-free test in `shape/policy_golden_test.go` that fails on a prompt without the rule (regression)
- The recorded prompt bytes follow the new prompt: the existing `TestPromptsAreByteEqualToTheElixirFunctions` and `TestShaperPolicyGolden` with their goldens updated (existing)

## Feasibility
- none
