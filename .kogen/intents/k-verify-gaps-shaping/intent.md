# Supply proof in contract-check fixtures

Slug: k-verify-gaps-shaping
Model: not applicable — this repairs synthetic execution evidence, not product state
Acceptance:
- .kogen/intents/k-verify-gaps-shaping/acceptance.feature
- features/steps/k_verify_gaps_shaping_steps.js

Make shaping's synthetic contract-check fixtures supply proof for the exact candidate being checked, compatible with K-VERIFY-GAPS. Keep user-visible behavior unchanged. Prefer test-only changes; production changes require demonstrated reliance on unproved verification success. Preserve useful assertions and all configured checks.

Build on the post-13ff132-revert base. LAND_WITH kogen-verification candidate 5c77ad602e614b07fa54083328505bdbae86d49d (landing tree ec8da6b, Require proof for verification success); do not fetch that unpublished revision. The supplied verification diff is the pairing specification. A scratch replace is for local testing only, never a committed module replacement. The final combined landing check must run both components together.

In particular, repair stale evidence when ccCorrectedFeature changes `Given the number 4` to `Given the number 6`. Reports must describe the candidate at that readiness attempt, not merely exit zero. Cover native and external Shape, including the external session that ends without a result. Preserve refusal feedback, read-only review/confirmation sessions, correction publication, readiness acknowledgment and incomplete-review behavior. Audit synthetic model-success fixtures for zero-execution assumptions under the supplied model.go contract; repair demonstrated failures without weakening proof checks. No new CLI commands or flags, raised timeouts, live data or provider calls.

## Decisions
- User-visible behavior is unchanged; this is a paired test compatibility fix, test-only unless evidence requires otherwise. (source: request)
- Use synthetic local fixtures and the existing npm Cucumber/Quint setup; never fetch the unpublished candidate. (source: request)

## Source comparison
- Addressed: request Problem and Need, including TestContractCheckRefusesAConfirmedContradictionAndPublishesItsCorrection and TestContractCheckExternalShape's `one read-only session per review and confirmation; the refusal is the next round's feedback`, plus fresh-run not_ready failures at lines 676 and 713. Acceptance runs those existing native/external tests and independently compares every captured report's pickle/source step proof to the expected candidate. Existing assertions remain required.
- Addressed: supplied verification Intent/report.go changes require exact source/pickle/test-step/result coverage; shaping must supply truthful evidence rather than weaken verification. The acceptance observer focuses on the demonstrated stale candidate proof; the combined landing verification and full configured suite remain required for the complete paired contract, including duplicate/omitted results, hooks/retries and model execution inventory.
- Addressed: constraints retain slug k-verify-gaps-shaping, LAND_WITH candidate 5c77ad602e614b07fa54083328505bdbae86d49d, post-13ff132-revert base, synthetic-only testing, unchanged CLI and timeout bounds. No persistent dependency replace is authorized.
- Omitted from implementation scope, not contradicted: verification-owned validation changes, its independent acceptance and assembling-kogen follow-ups in the reference Intent. Do not copy those product changes into shaping. Historical main 01aa16c and candidate/landing identifiers are context, not fetched dependencies. No current production defect is established by the historical eu2 failure alone.

## Tests
- Corrected candidate readiness has exact current step proof: existing native correction/publication test plus independent captured-report comparison (regression)
- External read-only review/confirmation, next-round feedback and incomplete-session behavior remain passing with truthful proof: existing TestContractCheckExternalShape and captured-report comparison (regression)
- All success fixtures satisfy paired verification, including actual model execution where configured: affected synthetic tests and full configured checks on the combined landing tree (integration)
- User-visible behavior, CLI surface, limits and useful existing checks remain unchanged: existing full suite (existing)

## Feasibility
- Local probe ran the existing named native/external tests with go test -json on the installed dependency; they passed, confirming these tests exist and the runner's JSON pass events. This does not establish compatibility with the unpublished candidate. Repository inspection found setReport parses shpFeature while ccCorrectedFeature changes its first Given text.
- No external documentation lookup: this is a local synthetic-fixture repair against an explicitly supplied unpublished Go contract, not an uncertain external protocol.
- evidence: {"id":"proof-observer","kind":"observer","inputs":["features/steps/k_verify_gaps_shaping_steps.js"],"observers":["features/steps/k_verify_gaps_shaping_steps.js"],"checks":[{"name":"good","command":"\"$KOGEN_NODE\" features/steps/k_verify_gaps_shaping_steps.js good","exit":0,"output":"control accepted\n"},{"name":"bad","command":"\"$KOGEN_NODE\" features/steps/k_verify_gaps_shaping_steps.js bad","exit":1,"output":"control rejected\n"}]}
