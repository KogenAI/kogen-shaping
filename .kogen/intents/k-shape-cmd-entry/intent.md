# Add a standalone shaping command

Slug: k-shape-cmd-entry
Model: not applicable — this is plain wiring of the existing Shape workflow, not a new state machine
Acceptance:
- .kogen/intents/k-shape-cmd-entry/acceptance.feature
- features/steps/k-shape-cmd-entry__steps.js

Add the developer-only cmd/kogen-shape executable, buildable with `go build -o <binary> ./cmd/kogen-shape` against this repository's own go.mod. It invokes shape.Shape on a supplied checkout without assembling or invoking the kogen executable or requiring an adoption in the component repository. The target checkout still needs Shape's existing configuration, tools and authentication. Do not bypass its readiness, approval boundaries or provider requirements. No phase behavior or Kogen user-facing CLI changes, timeout increases, or new dependency on kogen are permitted. Existing component dependencies, including the existing kogen-loop module dependency, remain usable.

Interface: `kogen-shape --project <checkout> --slug <slug> --request <file|->`. Read UTF-8 request text from that file, or stdin for `-`; pass an absolute RequestPath for files and an empty RequestPath for stdin. The same invocation with answers resumes pending Shape questions through existing Shape semantics. Validate the documented ShapeInput slug requirement (3–48 lowercase letters, digits and dashes) and nonempty request at the adapter boundary. Use the caller environment and working directory, pass project and slug to ShapeInput, and construct verify.Tools from KOGEN_NODE, KOGEN_QUINT and KOGEN_CUCUMBER (their existing meanings). Do not introduce new tool downloads or authentication mechanisms. Wire normal interrupt handling with the existing sys.Session conventions; do not introduce a deadline or change existing budgets.

Stdout is exactly one JSON object followed by a newline, not interspersed with phase prose: `{ "exit_code": 0, "intent": { "path": "<absolute written intent.md path>", "content": "<exact UTF-8 file contents>" }, "error": null }`. Its schema is stable: exit_code is an integer matching the process exit status; intent is either that object or null; error is null or a nonempty diagnostic string. Route existing session stdout/progress/diagnostics to stderr without changing Shape behavior. On a successful Shape publication, report the newly written `.kogen/intents/<slug>/intent.md`. On questions, protected-edit reports, refusal or errors, intent must be null even if an old Intent exists. Preserve Shape's exit codes (in particular pending questions/existing Intent exit 5), using its error/exit conventions; for nonzero exits report an error string, including a useful outcome description when Shape returns a nonzero exit without a Go error. Adapter usage/input failures exit 2 with the same JSON envelope. Document building, the interface, tool/auth prerequisites, result schema and answering questions in README.md. Standalone Refine is explicitly excluded. No changes to existing CLI commands or phase prompts/policy are needed.

## Decisions
- Provide a developer entry point, machine-readable written Intent and exit status without Kogen assembly; preserve phase and user-facing CLI behavior, timeout limits and dependency boundaries. (source: request)
- Support Shape only, including answering its pending questions; standalone Refine is outside this feature. (source: answer 1)

## Source comparison
- addressed: request K-SHAPE-CMD-ENTRY, filed by kogen-delivery on main's ruling, 3 Nov 2026, entanglement audit F14, repository kogen-shaping, change_type feature (developer entry point, not user-facing CLI): the outcome and interface above add cmd/kogen-shape using the own-module build and existing shape.Shape.
- addressed: Problem states only kogen-loop/cmd/kogen-loop is a main and existing shape.Shape/shape.Refine expose exit/files/stdout; the new executable and typed JSON envelope remove assembled-binary scoring dependency while keeping Shape's actual effects and statuses observable. Acceptance executes the real command on a local configured Git checkout and compares the typed result to the actual publication, then verifies an existing-Intent refusal cannot report stale content.
- addressed: Need requires running on a given checkout and a typed result in addition to exit code; the flags, file/stdin inputs, exact schema, integration scenarios and README requirements implement those behaviors. JSON is selected from the request's example, not a change to phase output semantics.
- addressed: Constraints 'No timeout raised; no new dependency on kogen' and 'No change to the phase’s behaviour or to Kogen’s CLI' apply to the complete implementation; retain existing phase tests and configured checks. No scope is granted for policy golden updates, provider changes, or timeout adjustment.
- superseded: the Problem mentions Refine as well as Shape, but answer 1 explicitly selects Shape only, including pending answers. Refine is deliberately omitted from the executable, not from the preserved rationale.
- addressed: supplied README/testing rules require Go build/vet/race tests, existing private-module authentication and narrow real-boundary wiring evidence; use the existing Go dependencies and local scripted-provider wire conventions from shape/helper_test.go. No live provider integration change or paid research is necessary.

## Tests
- Own-module standalone command runs Shape and reports the exact publication as typed JSON with matching exit status, from both a request file and stdin: command acceptance on a local Git checkout with the existing synthetic provider protocol and real Cucumber readiness (scenario)
- Nonzero Shape outcomes do not report an existing stale Intent and retain exit semantics: repeat the successful command to observe existing-Intent exit 5 and null intent (scenario)
- Pending Shape questions can be answered through the adapter without changing phase semantics: add a command integration test using the existing scripted-provider question fixtures and a second invocation with answers (integration)
- Adapter input requirements, stdout JSON isolation and usage exit 2 follow the documented interface: add command-boundary tests for valid slug/request and invalid slug/empty request (scenario)
- Existing phase behavior, timeout budgets, policy and Kogen CLI remain unchanged: retain current phase tests and run configured build, vet, format and Go test checks (existing)

## Feasibility
- Local API inspection confirmed sys.NewWith and sys.Session output routing, and verify.Tools fields Node, Quint and Cucumber. Existing repository helper_test.go provides the synthetic Responses SSE and credential fixture used in acceptance. (probe: exit 0, 0.2 s)
- evidence: {"id":"result-observer","kind":"observer","inputs":["features/steps/k-shape-cmd-entry__steps.js"],"observers":["features/steps/k-shape-cmd-entry__steps.js"],"checks":[{"name":"good","command":"\"$KOGEN_NODE\" features/steps/k-shape-cmd-entry__steps.js --control good","exit":0,"output":"accepted\n"},{"name":"bad","command":"\"$KOGEN_NODE\" features/steps/k-shape-cmd-entry__steps.js --control bad","exit":1,"output":"rejected\n"}]}
