# Restore the shaper's submit-early pressure

Slug: shape-time-target-keeps-submit-early
Model: not applicable — the change is prompt text; Shape's time handling has no state of its own
Acceptance:
- .kogen/intents/shape-time-target-keeps-submit-early/acceptance.feature
- features/steps/shape-time-target-keeps-submit-early__steps.js
- features/steps/shape-time-target-keeps-submit-early__driver/run.js
- features/steps/shape-time-target-keeps-submit-early__driver/observe_test.go.txt

Since the prompts made Shape time a target and not a stop, the shaper keeps exploring and revising until the shape-replay release gate ends the run with `time_cap`, and the heads since then hold the release windows with `shape_replay_regression`. The native shaper prompt (shape/prompts/shaper.md, its "Time target" bullet) and the external-route guidance (shape/prompts/external-guidance.md) again tell the shaper, without any stop, to converge inside the usual target: submit a complete package in the first rounds; once the elapsed time stated in the turn is within about 3 minutes of the usual target or past it, submit the best complete package it has and spend later rounds only on the refusals Kogen names, with no new exploration of files it has not read; a late package is still checked by every ordinary gate.

No stop is reintroduced. No code path refuses, skips or ends a model request because of elapsed time; no remaining-of-budget figure is printed; the per-request time line stays byte for byte `Time: <elapsed> elapsed. Target: <N> minutes usual, <N+5> minutes upper end.` (elapsed as budgetText writes it); KOGEN_SHAPE_BUDGET_MINUTES keeps its meaning (guidance only, 5 to 45, default 900 s); the literal, contract and decision checks gain no time input; the shape_time event is unchanged in every field. The change is limited to shape/prompts/shaper.md, shape/prompts/external-guidance.md, the prompt goldens under shape/testdata that embed them (shape/testdata/prompts_golden.json and shape/testdata/shaper_policy_golden.json, and shape/testdata/force_ordinary_golden.json where it embeds the prompt, regenerated with `go test ./shape -update` and reviewed), and new model-free test files in package shape. No Go production file changes.

The exact prompt text the Build writes is:

- In shape/prompts/shaper.md the sentences `The target is advice, not a stop: shaping continues past it whenever it can make progress, and a late package goes through the same checks. Plan the work to fit it: gather what you need in few turns, with independent reads and probes together, and submit early rather than late. Correct refusals promptly.` are removed, and these are added in their place: `Crossing the target is never a stop, but converge inside it: plan the work to fit it, gather what you need in few turns, with independent reads and probes together, and submit a complete package in the first rounds, early rather than late. Once the elapsed time stated in the turn is within about 3 minutes of the usual target or past it, submit the best complete package you have and spend later rounds only on the refusals Kogen names, with no new exploration of files you have not read. A late package is still checked by every ordinary gate.`
- In shape/prompts/external-guidance.md the sentences `The target is advice, not a stop. Plan to submit a complete package in the first round, well inside the target, and use later rounds only to answer Kogen's feedback.` are removed, and these are added in their place: `Crossing the target is never a stop, but converge inside it: submit a complete package in the first rounds, well inside the target. Once the elapsed time stated in the round is within about 3 minutes of the usual target of {{budget}} minutes or past it, submit the best complete package you have and spend later rounds only on the refusals Kogen names, with no new exploration of files you have not read. A late package is still checked by every ordinary gate.`

The Build's commit message and decision record state these removed and added sentences, and that the Build cannot run the shape-replay release gate: the test that fails on main before the change and passes after it is TestTimeTargetPromptTellsShaperToConverge (run with `go test ./shape -run TestTimeTargetPrompt`); the release gate measures the rest on its next window.

## Decisions
- The shaper prompt and the external guidance state the converge-inside-the-target instruction in the exact sentences above, with no stop; the prompt sentences removed and added are named in this Intent and in the Build decision. (decision: D1, round 1) (evidence: shape/prompts/shaper.md; shape/prompts/external-guidance.md; shape/budget.go) (source: request)
- Crossing the target never refuses, skips or ends a model request; no remaining-of-budget figure is printed and the time line stays byte for byte as budgetLineFor writes it. (decision: D2, round 1) (evidence: shape/budget.go) (source: request)
- KOGEN_SHAPE_BUDGET_MINUTES keeps its meaning: guidance only, 5 to 45, default 900 s. (decision: D3, round 1) (evidence: shape/budget.go) (source: request)
- A required under 120 seconds literal weakened to 125 seconds stays refused after the target is crossed; the literal check gains no time input. (decision: D4, round 1) (evidence: shape/literal.go) (source: request)
- The shape_time event is unchanged in every field, local_check_ms included. (decision: D5, round 1) (evidence: shape/evidence.go) (source: request)
- The scripted fourth-request scenario uses two refused submissions then a read round, never three same-gate refusals, which the fixed retry limit ends. (decision: D6, round 1) (evidence: shape/retrybudget.go; shape/shape.go) (source: request)
- The acceptance observer is supplied and copied over a temporary test file by the acceptance driver, so acceptance depends on no test the Build must invent; the Build separately adds the model-free tests named under Tests. (decision: D7, round 1) (evidence: shape/budget_test.go; shape/local_check_time_test.go) (source: request)

## Source comparison
- Addressed: prompt text restores submit-early pressure on both routes (scenarios 1 and 2); the `time_cap` outcome and the `shape_replay_regression` hold of the replay are the symptoms, which the release gate measures.
- Addressed: no stop, no remaining figure, the time line byte for byte, four requests after 1500 s (scenario 3).
- Addressed: weakened literal still refused (scenario 4); shape_time event unchanged (scenarios 5 and 6).
- Addressed: release evidence statement (the failing test and the sentences changed) in the Build decision. Not addressed here: the shape-replay gate itself, its kit, its 15 minute cap and repetition count, which the Build cannot run and does not change.
- Omitted: kogen-coordinator, kogen-setup, kogen-loop and the CLI; no new flag, setting or environment variable.

## Tests
- The native shaper prompt contains the instruction to submit the best complete package once the stated elapsed time is within about 3 minutes of the usual target or past it, and to spend later rounds only on named refusals, and the external-route guidance contains the same instruction with the target filled in: add TestTimeTargetPromptTellsShaperToConverge in shape/time_target_prompt_test.go (unit)
- A scripted native shaping on the fake clock of three rounds of 500 seconds each (two refused submissions at one gate, then a read round) still makes a fourth request after 1500 seconds, and the time line of every request equals the line e9e6d943 prints: add TestTimeTargetPromptNoElapsedStop (scenario)
- The weakened literal check still refuses a draft that weakens "under 120 seconds" to 125 seconds after the target is crossed, exactly as before it: add TestTimeTargetPromptLiteralStillRefusedLate (unit)
- The shape_time event of a scripted shaping equals the golden recorded before the change in every field: add TestTimeTargetPromptShapeTimeEventUnchanged (integration)
- The prompt goldens under shape/testdata follow the new prompt text: update the goldens (existing)

## Feasibility
- none
