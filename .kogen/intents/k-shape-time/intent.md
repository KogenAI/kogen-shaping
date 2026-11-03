# Record shape time and keep shaper commands short

Slug: k-shape-time
Model: not applicable — the account is arithmetic over spans that one sequential shaping run already passes through, and the commands rule is fixed task text; no workflow state, retry or limit is added
Acceptance:
- .kogen/intents/k-shape-time/acceptance.feature
- features/steps/k_shape_time_steps.js
- features/steps/k_shape_time_test.go.txt

K-SHAPE-TIME (fix; owner, 31 Oct 2026: "a 90-minute shape is unacceptable"). `kogen intent shape` with the claude-code executor regularly takes tens of minutes, and up to about two hours. Most of that time is not the model planning. It goes to the shaper polling long-running commands with `sleep N; cat`, to long test or acceptance runs started during shaping, and to Kogen-side gaps that cannot be explained. They cannot be explained because the shaping record keeps no time for claude-code rounds: `executor_round` events carry no time, and `phase_duration` `model` is 0 for claude-code shapes. No command, flag, option or setting is added; the agreed CLI stays as it is.

This change, in kogen-shaping only (package `shape`):

1. **Commands rule for the claude-code shaper.** `externalNote` (shape/external.go), the session note every external Shape and Refine session receives after the shaper instructions, gains one new last line with exactly this text:
   `Commands: Kogen runs readiness on every submission (the configured setup, the acceptance command, the Quint model and the Feasibility checks) and returns its actual results to this session. Do not repeat that work yourself: do not run the configured setup or checks, the acceptance command or a whole test suite, and do not prototype the implementation. Run only short commands that answer a named concrete uncertainty, such as one focused test, a probe or a tool query, each finishing on its own within 5 minutes. Never start a command in the background and never wait or poll with sleep: when you would need a longer run, submit your complete candidate and use Kogen's results.`
   The 5 minutes is Kogen's existing probe limit (`verify.ProbeTimeout`), the bound the native shaper's commands already run under. The paragraph appears exactly once in the main session's task (`executor.Task.Brief.Stable`). The native shaper's instructions (`ShaperInstructions`) and native request bytes are unchanged, and `shape/testdata/force_ordinary_golden.json` is not regenerated. Kogen cannot interrupt or refuse a running Bash call from this repository (see Source comparison), so this rule is the requested change of the shaper's behaviour. The account below shows any command time that remains, as shaper tools.

2. **Round timing on every `executor_round` event.** This covers every external session of Shape and Refine: the main shaper, audit, decision, question, contract review and confirmation, and Refine. Each event keeps its existing fields and gains four more:
   - `session`: the base name of the directory Kogen gives that session (`executor.Task.Dir`). Examples: `executor-shape` for the main Shape session, `executor-contract-review-<n>`, `executor-contract-confirm-<n>`, `executor-decision-<n>`, `executor-question-<n>`, `executor-audit`, `executor-refine`.
   - `started_ms`: Kogen's clock right before the executor's `Session.Round` call, in milliseconds since the run's start (the shaping record's `Evidence.Millis`).
   - `wall_ms`: from then until that call returned.
   - `model_ms`: the round's model time as Claude Code reports it. Kogen reads the session's retained raw stream of that round, `<session directory>/round-<n>.jsonl`, where `<n>` is the session's round number (1 for its first `Round` call). It takes `duration_api_ms` from the last line that is a JSON object with `"type":"result"` and a non-negative integer `duration_api_ms`. It is `null` when the file is missing or unreadable, or has no such line. It is never estimated.
   A round that fails still records no `executor_round` event (unchanged), but its span still counts below.

3. **The shape time account.** Every `kogen intent shape` run that opened its shaping record (`.kogen/local/shaping/<slug>/events.ndjson`) ends that record with one event. It comes after the existing `usage_totals` and `phase_duration` events and immediately before `session_end`:
   `{"event":"shape_time","elapsed_ms":E,"shaper_ms":S,"shaper_model_ms":M,"shaper_tools_ms":T,"checks_ms":C,"readiness_ms":R,"gap_ms":G}`
   It has exactly these keys, and every known value is a whole number of milliseconds.
   - `E` is the run's elapsed time, the same value as `session_end` `elapsed_ms`.
   - Shaper spans, claude-code: each round of the main Shape session (`executor-shape`), from right before its `Session.Round` call until that call returns, whether or not it succeeded. These are the `started_ms`/`wall_ms` spans of point 2. `S` is their total. `M` is the sum of their `model_ms`, or `null` when any of those rounds has none (including a failed round). `T = S − M`, or `null` when `M` is null.
   - Shaper spans, native: each request of the main conversation, from right before it is sent until the response or error comes back (transport retries included), is a model span. Each tool call of the main conversation other than `submit` (`read_file`, `list_files`, `search_text`, `view_image`, `probe`, `documentation`, `propose_files`, `shape_batch`), from its start until its output is ready, is a tool span. `M` is the total of the model spans, `T` the total of the tool spans, and `S = M + T`.
   - Check spans are Kogen's own checking work:
     - the existing `setup` and `preparation` phases;
     - the request audit, when enabled;
     - the handling of every submission, from receiving it (the `submit` call or the external session's structured result) until its feedback or outcome is decided. This includes validation, the question, decision and contract checks with their sessions and calls, the step self-check, the Feasibility checks, readiness with its gate and control run, and writing the candidate;
     - for claude-code, the custody comparisons before and after each main round (the Git execution-state pin, the checkout identity, and removal of untracked files the round created).
     `C` is the total length of the check spans, with overlaps counted once.
   - `R` is the existing readiness time: the same value as the `phase_duration` event with `phase` `readiness`.
   - Shaper spans and check spans never overlap, so `G = E − S − C` (never negative). `G` is everything else Kogen did: starting an external session, the native working window, manifests, the questions file, and writing outputs.
   `kogen intent refine` records no `shape_time` event and prints no time line. Its record changes only by the point 2 fields.

4. **The time line.** After the shaping record is finished, Shape writes one more progress line to stderr. It is Shape's last stderr line on every exit of a run that opened its shaping record: ready, questions, blocked, not ready or stopped:
   `kogen: shape time <E>: shaper <S> (model <M>, tools <T>), Kogen checks <C> (readiness <R>), gaps <G>`
   Each value is in seconds with one decimal, rounded half up from milliseconds: `tenths = (ms + 50) / 100` in integer division, printed as `<tenths / 10>.<tenths % 10> s`. So 1250 ms prints `1.3 s`, 1249 ms prints `1.2 s`, and 0 ms prints `0.0 s`. A `null` value prints `unknown`. Example: `kogen: shape time 6.1 s: shaper 1.9 s (model 0.5 s, tools 1.4 s), Kogen checks 3.0 s (readiness 2.1 s), gaps 1.2 s`. Stdout is unchanged, including the ready output and its usage line `shape <model>/<effort> claude-code <n> round(s) wall_ms=<ms>`.

Unchanged and out of scope:
- the existing events (`phase_duration` with its `model` and `shape` phases, `usage_totals`, `session_end`, `executor_summary`, `executor_round_started`/`_ended`, `executor_tool`);
- every gate and its feedback text, readiness, the contract, decision and question checks;
- the approval card, `kogen intent approve` and the Build;
- the native shaper's instructions and request bytes;
- the executor (kogen-executor) and readiness (kogen-verification);
- the configured commands;
- `.kogen/project.yaml`.

What other repositories must change is a follow-up item, not part of this one (see Source comparison).

## Acceptance harness
- `features/steps/k_shape_time_steps.js` writes a go overlay that adds `features/steps/k_shape_time_test.go.txt` as `shape/k_shape_time_acceptance_test.go` (paths relative to the checkout root). It builds once per Cucumber run with `go test -c -vet=off -overlay <tmp>/overlay.json -o <tmp>/shape.test ./shape`, then runs `<tmp>/shape.test -test.run ^TestKShapeTimeAcceptance$ -test.count=1 -test.v -test.timeout=120s` in `shape/`.
- `KST_MODE` selects the run:
  - `external`: the is-even fixture with the shape role `executor: claude-code`, run by the scripted `shapeExec`;
  - `native`: the loopback provider: read README and submit, then acknowledge.
- Further variables:
  - `KST_ROUNDS` lists the two external main rounds as `<sleep ms>:<duration_api_ms|none>`. Each round sleeps, then writes `.kogen/local/shaping/is-even/executor-shape/round-<n>.jsonl` with an init line and a result line carrying that `duration_api_ms`; `none` writes no file. Round 1 submits the package and round 2 acknowledges the red readiness.
  - `KST_CONTRACT_MS` makes each external contract review round sleep, then answer no findings.
  - `KST_ACCEPT_SECONDS` adds `sleep <n>` to the fixture's `accept.sh` after its `esac` line.
- Each run prints one `KST <json>` line with:
  - `exit` and `stop`;
  - `stdout` lines;
  - `time_lines` (stderr lines starting `kogen: shape time`) and `stderr_last`;
  - `accounts` (the `shape_time` events) and `after_account` (the event after it);
  - `session_end`, `readiness_phase` (the readiness `phase_duration` values) and `rounds` (every `executor_round` event);
  - `brief`: the `Brief.Stable` of the session whose `Task.Dir` base name is `executor-shape`.
- The Build must not create a file at the overlay path. It keeps the shape test helpers the harness calls, with their names and behaviour:
  - `newShpFx`, `useExternalShape`, `shapeWith`;
  - `shapeExec` with its `rounds`, `contract` and `starts` fields (rounds receive the session's round number);
  - `shpSubmit`, `shpAck`, `shpFiles`, `shpFeature`, `shpSlug`, `shpNamed`;
  - `read`, `write`, `git`, `shape`, `events`, `Stdout` and `Stderr`.

## Decisions
- Time spent inside a shape is attributable afterwards from what Kogen keeps: the shaping record ends with one account of shaper model, shaper tools, Kogen checks (with readiness) and gaps, and Shape states the same account as its last stderr line, in seconds with one decimal, with `unknown` where a value was not reported (source: request)
- The claude-code shaper is told to leave readiness to Kogen, to run only short commands of at most 5 minutes, and never to start commands in the background or wait or poll with sleep (source: request)
- No command, flag, option or setting is added; the ready output, approval card, readiness, approval and Build stay as they are (source: request)

## Source comparison
- Addressed, Problem (polling with `sleep N; cat`, long test or acceptance runs started during shaping) and Need "no shape spends most of its time waiting on commands or polling": point 1. Acceptance: "The claude-code shaper is told to keep commands short and never poll". Supporting evidence: in K-GO-TEST-CRITICAL-PATH, 19 polls took 73 min and the largest call was 600 s; in S-BROWSER-LOAD-WAIT, cucumber runs took 590, 560 and 400 s; in K-CHECKTIME-SHARD, polls took 540, 420 and 300 s. A retained claude-code record in this host's k-models-file-shaping checkout shows the same pattern: Bash calls of 401, 368 and 204 s that edited and tested copies of the implementation under /tmp.
- Addressed, Problem (unexplained Kogen-side gaps) and Need "time spent inside a shape is attributable afterwards (model, tools, Kogen checks, gaps) from what Kogen keeps": points 2 to 4. Acceptance: "A claude-code shape records its model, tool, check and gap time", "Model time that a round does not report stays unknown" and "A native shape records the same account". The time line also survives in a caller's log when the shaping record does not; the request notes that run records were mostly not kept and that two cases had no transcript.
- Addressed, the measured categories: the account's fields are the categories the request measured by hand. These are model time (S-MEDIA-INTAKE: API time 1451.6 s of a 1460.1 s round), tool time (K-GO-TEST-CRITICAL-PATH: Bash 87.7 of 115.6 min), Kogen checks with readiness (S-CANCELFLAKE-4 and K-CHECKTIME: 66% and 88% in readiness; a 1535 s interval; 6.9 min of readiness in S-BROWSER-LOAD-WAIT) and gaps (S3 shape2: 4451 s; S-BROWSER-LOAD-WAIT: 413 s). The quoted figures and the 113-shape table are context, not executable acceptance. `SHAPE-TIMES.md`, named in the request, is not in this checkout.
- Addressed, Need "output quality does not drop: same scenarios, checks and approval card contents" and "nothing else about building or approving changes": no gate, submission rule, readiness, approval card, ready output or Build behaviour changes, and the native instructions are byte-identical. Existing tests and the force golden stay green.
- Addressed, "No new command, flag or option; the agreed CLI stays as it is": nothing is added. The time line is progress output on stderr.
- Decided: the model time of a claude-code round comes from the retained raw stream's result line (`duration_api_ms`). The executor reports no time in its events, and that stream already sits in the Kogen-owned session directory (`executor.Task.Dir`). An observed Claude Code 2.1.312 result line in a retained record on this host has `duration_ms` 3160386 and `duration_api_ms` 1368516. Tools are the rest of the round, not a separately measured value. When the line is absent, the value is `null`, never 0 or a guess.
- Decided: claude-code check sessions (contract review, decision and question checks, audit) count as Kogen checks, not shaper time. The custody comparisons around main rounds also count as checks.
- Decided: the shared `externalNote` also reaches Refine's external session, which runs the same readiness. Refine records the new `executor_round` fields but no account; the request concerns `kogen intent shape`.
- Follow-up items in other repositories, not part of this Intent:
  - kogen-executor: today it hooks only Write/Edit/MultiEdit/NotebookEdit before use and observes Bash only after it ran, so it cannot refuse a Bash call. The follow-up is to enforce a per-command bound for the shape role and refuse background starts and sleep polling there, and to report model time and per-tool spans in its own events so Kogen no longer reads the raw stream.
  - kogen-verification: shorten readiness when the account shows it dominating.
  - the kogen caller: retain shaping records.
- Owner's release acceptance, outside this Build: "Before/after on the same Intents with the shipped binary, same model and effort (fair comparison), including at least one Intent whose shape ran tests during shaping". These are live paid model runs (testing rules section 5, item 4). The `shape_time` account and the time line supply the split for that comparison. This Intent's acceptance is synthetic only.

## Tests
- The claude-code shaper's task contains the commands paragraph exactly once: acceptance "The claude-code shaper is told to keep commands short and never poll". Also extend the task-content check of `TestClaudeLineupShapeSubmission` "readiness feedback and exact red acknowledgment stay with one session" with the paragraph (integration).
- Refine's external task contains the same paragraph: extend `TestClaudeLineupRefinePreservation` with a check of the started task (integration).
- Every `executor_round` event carries `session`, `started_ms`, `wall_ms` and `model_ms`, with `model_ms` from the last result line of `round-<n>.jsonl` and `null` when that file is missing, has no result line or has a non-integer `duration_api_ms`: add a focused test in shape/lineup_shape_test.go whose scripted rounds write those files (integration)
- The `shape_time` event has exactly its keys, `elapsed_ms` equal to `session_end` and `readiness_ms` equal to the readiness `phase_duration`, `shaper_ms` as the total of the shaper spans, `checks_ms` with overlaps counted once, `shaper_model_ms` null when any main round lacks model time, `shaper_tools_ms` as shaper minus model, `gap_ms` as elapsed minus shaper minus checks, and sits right before `session_end`: add a unit test in shape/evidence_test.go using its test clock (unit)
- The time line format: 1250 ms gives `1.3 s`, 1249 ms gives `1.2 s`, 0 gives `0.0 s`, and null gives `unknown`. Add a unit test of the line (unit).
- A claude-code shape attributes its round, model, tool, check (readiness and contract review sessions) and gap time, and keeps unknown model time unknown: acceptance scenarios 1 and 2 (scenario).
- A native shape records the same account with known model and tool time: acceptance scenario 3 (scenario).
- The time line is also Shape's last stderr line, and the account is recorded, when Shape ends with questions (exit 5): add an integration test (integration).
- Ready output, usage line, approval card, native request bytes and the existing phase events are unchanged. The existing tests cover this: `TestClaudeLineupShapeSubmission`, the shape_test.go phase checks, the evidence_test.go phase tests, the approve tests and the force golden (existing).

## Feasibility
- Session observation (my own reads on this host, not a Kogen probe record): the retained claude-code shaping record `.kogen/local/shaping/k-models-file-shaping/` of a 54-minute shape (`phase_duration` `shape` 3241128 ms) records `phase_duration` `model` 0 and `executor_round` events without any time. Its `executor-shape/round-1.jsonl` has assistant and user lines with `timestamp`s and a final result line with `duration_ms` 3160386 and `duration_api_ms` 1368516 (107 turns). Tool spans derived from the timestamps total about 1793 s, the largest being Bash calls of 401, 368 and 204 s. The executor emits `executor_round_started`, `executor_tool` (at each tool result, with no start) and `executor_round_ended` in real time, without durations. Its hooks run before Write/Edit/MultiEdit/NotebookEdit only and after Bash, and it already sets `CLAUDE_CODE_DISABLE_BACKGROUND_TASKS=1`.
- Unchanged-tree run of the submitted harness from a scratch copy of HEAD (outside the checkout), with the configured acceptance command: all 4 scenarios fail on the missing feature, in about 20 s including the overlay compile. Scenarios 1 to 3 fail at "the shaping record ends with one shape time account" (no `shape_time` event), after the ready output passed. Scenario 4 fails at the paragraph check (0 occurrences in the 36437-byte task). On that tree the external run records 2 main rounds and 1 `executor-contract-review-1` round, with readiness about 2.1 s with a 1-second acceptance. The observer controls printed `accepted` (exit 0) for good and `rejected` (exit 1) for bad.
- evidence: {"id":"shape-time-observer","kind":"observer","inputs":["features/steps/k_shape_time_steps.js","features/steps/k_shape_time_test.go.txt"],"observers":["features/steps/k_shape_time_steps.js","features/steps/k_shape_time_test.go.txt"],"checks":[{"name":"good","command":"\"$KOGEN_NODE\" features/steps/k_shape_time_steps.js good","exit":0,"output":"accepted\n"},{"name":"bad","command":"\"$KOGEN_NODE\" features/steps/k_shape_time_steps.js bad","exit":1,"output":"rejected\n"}]}
