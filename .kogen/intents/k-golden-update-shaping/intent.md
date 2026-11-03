# Add a test-only golden update flag

Slug: k-golden-update-shaping
Model: not applicable — this is test-fixture regeneration wiring, not application state
Acceptance:
- .kogen/intents/k-golden-update-shaping/acceptance.feature
- features/steps/k_golden_update_steps.js

Implement one shared Go test-only `-update` flag, supporting `go test ./... -update`. Regenerate both repository goldens from current synthetic execution results: `shape/testdata/force_ordinary_golden.json` and `shape/testdata/init/mix-golden.json`. No production CLI flag, command, config key or environment switch is added. No timeouts are raised. Preserve all other assertions and ordinary comparison semantics. Tests must still reject drift without the flag and must never write without it. Update mode must not suppress unrelated failures.

For force captures, collect all four existing shape/refine × responses/chat ordinary sessions using the existing normalization and capture format; write the complete map, not just the last case. Preserve the forced-tool enforcement assertions. For mix validation, preserve the synthetic text/check inputs and case order; replace only expected eligibility with actual Go validation results, retaining lowercase JSON field names. Do not rely on the old Elixir generator to refresh Go expectations. Use deterministic formatting, trailing newline and checked write errors. Keep flag registration and writing machinery in test-only Go files, shared by the golden tests. Do not introduce production dependencies for this.

Document in README Testing: run `go test ./... -update` only after deliberate behavior changes or to resolve golden conflicts on the combined tree; inspect the diff, then run `go test ./...` and the normal full checks. Never hand-merge hashes. Update test comments to explain precisely what each golden stores and its provenance: the force golden originally recorded ordinary wire bytes before enforcement existed; it remains a deliberate compatibility baseline, not a routinely refreshed snapshot. Mix inputs came from the Elixir compatibility fixture, while explicit updates record current Go results. No live user data or providers may be used.

## Decisions
- Use `go test ./... -update` as the single documented test-only interface; normal tests remain read-only and strict. (source: request)
- Keep deliberate compatibility baselines reviewable rather than exempting them from explicit updates. (source: request)

## Source comparison
- Addressed: Request Need and repository inventory map to the flag, two complete regeneration paths, strict ordinary mode, provenance comments and README instructions above and the executable scenarios below.
- Addressed: Request Problem, Evidence and Why now describe hash conflicts after the 2 Nov K-SHAPE-DECIDE-NON-UX and K-MODELS-FILE landings and the 3 Nov 13ff132 rebuild. Acceptance corrupts recorded expectations, regenerates from execution, and requires a subsequent ordinary pass; no historical builds or rebuild durations are fabricated as tests.
- Omitted as out of repository: examples `build/testdata/k-revod-gate/parts-requests.json`, `build/testdata/context-external-golden.json` and `reviewretry/*.wire.json` motivate the shared convention but are not present in this component's requested inventory. No cross-repository edits are required.
- Superseded: the current mix test treats eligibility as an exclusively Elixir-owned immutable oracle; explicit updates now deliberately capture current Go output while retaining the synthetic inputs. Without the flag its comparisons remain unchanged.
- Addressed: synthetic fixtures only, no new binary option, and no raised timeouts apply to all implementation work. Studio issue 22c28a45-a727-4ba6-8a19-0e7731a888a3 is context, not a new runtime integration.

## Tests
- Both goldens reject expectation drift without writing: corrupt each expectation in a temporary checkout and run its existing test (integration).
- Explicit update regenerates actual results and a following ordinary run passes: run each existing test with `-update`, check its output file changed, and rerun without the flag (integration).
- Updates are deterministic and preserve fixture inputs: repeat update and compare bytes; compare mix inputs and force case keys before/after (integration).
- Existing compatibility and enforcement assertions remain: retain TestShapeForceWireCompatibility and TestInitMixVerificationMatchesElixir assertions in ordinary mode (existing).
- Document deliberate use and keep the option test-only: review README commands and provenance comments, run configured build/vet/full tests (existing).

## Feasibility
- Existing focused mix test passed (probe: exit 0, 16.0 s); verbose run confirmed Go's real pass marker (probe: exit 0, 22.0 s). Both existing selected tests passed together (probe: exit 0, 55.9 s).
- evidence: {"id":"golden-observer","kind":"observer","inputs":["features/steps/k_golden_update_steps.js"],"observers":["features/steps/k_golden_update_steps.js"],"checks":[{"name":"good","command":"node features/steps/k_golden_update_steps.js --control good","exit":0,"output":"accepted\n"},{"name":"bad","command":"node features/steps/k_golden_update_steps.js --control bad","exit":1,"output":"rejected\n"}]}
