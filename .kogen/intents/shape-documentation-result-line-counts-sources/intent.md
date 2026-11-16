# Count sources in the documentation result line

Slug: shape-documentation-result-line-counts-sources
Model: not applicable — the change is one progress line derived from a retained lookup row; there is no state to model
Acceptance:
- .kogen/intents/shape-documentation-result-line-counts-sources/acceptance.feature
- features/steps/shape-documentation-result-line-counts-sources__steps.js
- features/steps/shape-documentation-result-line-counts-sources__driver/harness_test.go.txt

The finished progress line of a documentation lookup said only `kogen: documentation result: search retrieved`, so a search that returned no sources looked the same as one that worked. `DocumentationProgress` in shape/documentation.go now names the source count for a search whose result row carries a sources list and has `status: retrieved`: `kogen: documentation result: search retrieved (0 sources)`, `(1 source)`, `(2 sources)`, `(9 sources)`; the count is the length of the row's `sources` list. A fetch, and any row whose status is not `retrieved`, keep today's finished line exactly, and the started line is unchanged.

Only the progress line changes: not the retained row, the `documentation_lookup` event, `documentation.ndjson`, the provenance gates or which drafts pass, and no configuration is added. A focused model-free Go test is added in the shape package next to shape/discovery_test.go, using its existing helpers: a search row with two sources prints `(2 sources)`, one with none prints `(0 sources)`, a fetch row prints the line unchanged.

## Decisions
- The count is the length of the retained row's `sources` list, singular `(1 source)` and plural otherwise including `(0 sources)`; the suffix applies only to a search with `status: retrieved` (decision: D1, round 1) (evidence: shape/documentation.go) (source: request)
- Fetch lookups and rows with another status such as `unsupported` keep their current finished line, and the retained row, event and provenance gates are untouched; no setting is added (decision: D2, round 1) (evidence: shape/documentation.go; shape/discovery_test.go) (source: request)
- Acceptance drives RetainDocumentation and DocumentationProgress through a Go test harness copied into the shape package by a Go build overlay (an unwritten test is not run by name) and reads the lines from a JSON file the harness writes (decision: D3, round 1) (evidence: shape/discovery_test.go; features/steps/approve-static-reuse-line-explains-unknown__steps.js) (source: request)

## Source comparison
- Observed bare line, from the request: `kogen: documentation result: search retrieved` for searches that returned 0, 9, 0 and 9 sources; superseded by the counted line in acceptance.feature scenario "A retrieved search names how many sources it returned".
- Expected lines `(0 sources)`, `(1 source)`, `(9 sources)` from the request: addressed with 0, 1 and 2 sources in the same scenario; the count is the same rule for any number.
- Fetch and non-retrieved rows keep today's line: addressed in scenario "Other lookups and the started line keep their text".
- No change to the retained row, event, documentation.ndjson, provenance gates or configuration: omitted by design.

## Tests
- A retrieved search names its source count, a fetch and a non-retrieved row keep the line: add a focused model-free test in shape/discovery_test.go using its scripted documentation helpers (unit)
- The existing progress assertions keep passing: TestShapeRecordsEachDocumentationLookupAsProgressEventAndRecordWithoutTheBody and TestADocumentationLookupIsRetainedWithoutTheDocumentBody (existing)

## Feasibility
- none
