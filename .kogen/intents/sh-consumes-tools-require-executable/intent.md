# Use executable tool fixtures in consumer tests

Slug: sh-consumes-tools-require-executable
Model: not applicable — two consumer tests change their fixtures and one assertion; there is no state or interacting rules
Acceptance:
- .kogen/intents/sh-consumes-tools-require-executable/acceptance.feature
- features/steps/sh-consumes-tools-require-executable__steps.js
- features/steps/sh-consumes-tools-require-executable__harness.js
- features/steps/sh-consumes-tools-require-executable__control.js

This is the consumer row paired with kogen-verification's Build broken-kogen-install-blamed-on-intent (candidate 7b03b35, Build 4677e906). It lands first; the producer is then re-shaped on top of it. That Build changes the exported `verify.Tools.Require` on purpose: node must be executable, not just a regular file, and the refusal text changes from "node is not available at X; reinstall Kogen" to "node is not available: X …". It also adds `RequireInstall`. Its consumer contract check (5 Nov 09:51Z) refused it because of two new failures in this repository:

```
kogen-shaping TestReadinessCheckAndKeptEvidence readiness_test.go:89: verify.Readiness.Check: environment/tool_missing: node is not available: node does not exist; reinstall Kogen
kogen-shaping TestToolsFromInstall tools_test.go:58: verify.ToolsFromInstall: without a node-path file Node is that file's path: …
```

After this change, `TestReadinessCheckAndKeptEvidence` (`consumes/kogen-verification/readiness_test.go`) and `TestToolsFromInstall` (`consumes/kogen-verification/tools_test.go`) pass against the kogen-verification this repository pins today (`github.com/KogenAI/kogen-verification v0.0.0-20261105003813-ac4f6a9a4877`) and against one with that new `Tools.Require` rule and wording. Only those two test functions, and the doc comment of `TestToolsFromInstall`, change. No other test, no pin (`go.mod`, `go.sum`), no README and no production code changes. Every existing test stays green.

Candidate 7b03b35 is not in the module cache and is not on the public main branch, so acceptance uses a stand-in. The stand-in is a scratch copy of the pinned module, not that candidate's source. It reproduces both reported failures on the unchanged tests:
- `Tools.Require` refuses a Node that is missing or not an executable regular file. A missing Node uses the reported detail `node is not available: node does not exist; reinstall Kogen` (class `environment`, reason `tool_missing`, exit 3). A present non-executable Node is refused as `node is not available: <path> is not executable; reinstall Kogen`.
- `Readiness.Check` calls `Tools.Require` first when the tools are not the zero `verify.Tools{}`. That is the only Check in these tests that sets tools, and the only one the report shows failing. The pinned Check calls `Require` only when the Intent model is not `None`, which is why `/tools/node` passes today.
- Without a readable `node-path` file, `ToolsFromInstall` gives an empty Node. The reported got value was truncated; empty is the stand-in, not an observed producer field. The readable `node-path` text is still returned trimmed, including when that path is not a file (tools_test.go:65 was not in the report).

Required edits, in `consumes/kogen-verification/`, package `consumes`. Standard library only. Every new file is under `t.TempDir()`. Do not use a new `verify` identifier. Do not use `write` for an executable: `write` saves mode `0o644`.
- `TestReadinessCheckAndKeptEvidence`: replace `r.Tools = verify.Tools{Node: "/tools/node", Quint: "/tools/quint"}`. In a fresh `t.TempDir()`, create two regular files named `node` and `quint`, each with content `#!/bin/sh\n` and mode `0o755`, and set `r.Tools = verify.Tools{Node: <node>, Quint: <quint>}`. Leave `Cucumber` unset. The setup-trace expectation becomes `<node>|<quint>|candidate\nsecond\n` with those absolute paths. The failure text still says the tools reach setup as `KOGEN_NODE` and `KOGEN_QUINT`. Every other assertion of the test stays as it is.
- `TestToolsFromInstall`, case without a `node-path` file: `Quint` must still equal `<root>/lib/quint/node_modules/@informalsystems/quint/dist/src/cli.js` and `Cucumber` must still equal `<root>/lib/cucumber/node_modules`. `Node` must name no executable regular file: `os.Stat` fails, the file is not regular, or `Mode().Perm()&0o111 == 0`. Do not pin the Node string. That passes the pinned producer (Node is the unread `node-path` file's path, which does not exist) and the stand-in (empty Node). Each failure message starts with `verify.ToolsFromInstall:` and shows the value it got.
- `TestToolsFromInstall`, `node-path` case: in a separate `t.TempDir()` outside the install root, create a real executable (content `#!/bin/sh\n`, mode `0o755`). Write its absolute path followed by `\n` to `<root>/node-path`, instead of `/opt/node/bin/node\n`. The whole result must equal `verify.Tools{Node: <that path>, Quint: <as above>, Cucumber: <as above>}`, which proves the text is trimmed.
- Update the doc comment of `TestToolsFromInstall` so it no longer says that an unreadable `node-path` makes Node that file's path. Say that, without a readable `node-path`, Node names no executable file, and that Quint and Cucumber are still the paths under `installRoot/lib`.

## Decisions
- Both tests pass against the pinned kogen-verification and against one with the new `Tools.Require` rule and wording. (source: request)
- Tool fixtures are real executables: mode `0o755` regular files with content `#!/bin/sh\n`, not the missing paths `/tools/node`, `/tools/quint` or `/opt/node/bin/node`. (source: request)
- Refusal checks keep class, reason, exit and the named tool, and do not pin the sentence around the path. These two tests are success checks, not refusal checks, and none is added. The reported missing-node detail is `node is not available: node does not exist; reinstall Kogen`; it names `node` and does not contain the path. A refusal assertion in these two tests, if one were written, would check class `environment`, reason `tool_missing` and exit 3, and would not compare the whole sentence or require a path that detail does not contain. (source: request)
- No other test, pin (`go.mod`, `go.sum`) or behaviour changes, and every existing test stays green. `RequireInstall` is not used by this repository and is not tested here. (source: request)

## Source comparison
- Addressed: both named tests pass against the pin `github.com/KogenAI/kogen-verification v0.0.0-20261105003813-ac4f6a9a4877` and against the new executable-node rule. Scenarios "Both tests pass against the pinned kogen-verification" and "Both tests pass against kogen-verification with the new Tools.Require rule and wording"; required edits above.
- Addressed: tool fixtures are real executables. Required edits replace `/tools/node`, `/tools/quint` and `/opt/node/bin/node` with `0o755` files of content `#!/bin/sh\n`.
- Addressed, with a conflict on the path: refusal checks keep class `environment`, reason `tool_missing` and exit 3, and do not pin the sentence. The request also says they keep the path. The reported missing-node detail `node is not available: node does not exist; reinstall Kogen` does not contain the path, so these success tests add no refusal assertion and do not require that path. Decisions, third bullet.
- Addressed: no pin change. Scenario "The module pins are unchanged" names the four `go.mod` requirements in order, no replace directive, and go.sum SHA-256 `ba20dbb2a5ce87b0077fc8b5843cbe58efd06ea32feb63fcf724c06a92c93229`.
- Addressed: no other test or behaviour change; existing tests stay green. Only the two functions and the `TestToolsFromInstall` doc comment change. Preserved assertions are the setup trace, the lib paths, and the trimmed `node-path` text. The Scenario Outline rejects a producer that alters that text, drops `KOGEN_NODE`, or resolves a missing `node-path` to `/bin/sh`.
- Omitted: `RequireInstall`, candidate `7b03b35` and Build `4677e906` as code. This repository does not call `RequireInstall`. The ids are recorded above; the producer is re-shaped later, on top of this row.
- Superseded: the unread `node-path` file's own path as the required Node value. The report shows tools_test.go:58 failing on that equality. The new contract is "Node names no executable regular file", which the pinned value and an empty Node both meet.

## Tests
- A readiness check with given tools passes under the pinned producer and under the new executable-node rule, and those tools reach setup as KOGEN_NODE and KOGEN_QUINT: update TestReadinessCheckAndKeptEvidence to use real executable Node and Quint fixtures (integration)
- ToolsFromInstall without a node-path file keeps the lib paths, and its Node names no executable regular file: update TestToolsFromInstall (integration)
- ToolsFromInstall with a node-path file returns the trimmed path of a real executable fixture: update TestToolsFromInstall (integration)
- Module pins are unchanged: acceptance scenario "The module pins are unchanged" (scenario)
- Every other existing test stays green: the configured go-test check (existing)

## Feasibility
- The pinned module's `verify/env.go` and `verify/readiness.go` contain each stand-in patch needle once. Against that stand-in the unchanged tests fail as reported, including the exact Check detail `node is not available: node does not exist; reinstall Kogen`. The observer accepts the pinned run and rejects the altered node-path text. (probe 1: exit 0, 5.9 s)
- evidence: {"id":"consumer-tests-observer","kind":"observer","inputs":["features/steps/sh-consumes-tools-require-executable__control.js"],"observers":["features/steps/sh-consumes-tools-require-executable__harness.js"],"checks":[{"name":"good","command":"\"$KOGEN_NODE\" features/steps/sh-consumes-tools-require-executable__control.js good","exit":0,"output":"TestReadinessCheckAndKeptEvidence: pass\nTestToolsFromInstall: pass\naccepted\n"},{"name":"bad","command":"\"$KOGEN_NODE\" features/steps/sh-consumes-tools-require-executable__control.js bad","exit":1,"output":"TestReadinessCheckAndKeptEvidence: pass\nTestToolsFromInstall: fail\nrejected\n"}]}
