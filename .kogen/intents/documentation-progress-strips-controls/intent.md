# Convert control characters in documentation

Slug: documentation-progress-strips-controls
Model: not applicable — the change is a stateless string transformation of one progress line
Acceptance:
- .kogen/intents/documentation-progress-strips-controls/acceptance.feature
- features/steps/documentation-progress-strips-controls__steps.js
- features/steps/documentation-progress-strips-controls__driver/main.go

The started progress line of a documentation lookup (`DocumentationProgress` in `shape/documentation.go`) shows the lookup's reference (its url, else its query). Today only carriage returns and line feeds in that reference become spaces. After this change every control character in the reference becomes one space, so the reference stays on one safe, printable line, and ordinary text (spaces and printable Unicode) is preserved unchanged. The existing replacement of a line break by a space stays. The change replaces the carriage-return/line-feed-only replacement; nothing else changes. Out of scope: documentation lookup results, retained evidence formats and provider behavior.

## Decisions
- Every control character of the displayed reference (the Unicode control category, which covers C0 characters such as tab, NUL, ESC, CR, LF, DEL and the C1 range) is replaced by exactly one space; ordinary spaces and printable Unicode are left untouched, and no runs of spaces are collapsed, so a CR followed by LF still yields two spaces and a lone LF one. (evidence: shape/documentation.go; shape/discovery_test.go) (source: request)
- The `finished` progress line, the retained row and the `-` placeholder for a missing reference are unchanged. (decision: D2, round 1) (evidence: shape/documentation.go) (source: request)
- Acceptance observes the behavior through a small Go program that calls the exported `shape.DocumentationProgress` on a row read from stdin, because the function is the only entry point that builds the line. (decision: D3, round 1) (evidence: shape/documentation.go; shape/shape.go) (source: request)
- Every control character of the displayed documentation progress reference is replaced by exactly one space; ordinary spaces and printable Unicode are untouched and spaces are not collapsed. (decision: D1, round 1) (constraint: finished line, retained row and '-' placeholder unchanged) (evidence: shape/documentation.go; shape/discovery_test.go) (assumption: Unicode line/paragraph separators (U+2028/2029) are not control-category characters and are left as is; the request names only control characters.) (source: request)

## Source comparison
- Request: a reference with carriage return, line feed, tab, NUL and ESC yields a single progress line with each converted to a space — addressed by scenarios 1 and 2 of acceptance.feature.
- Request: ordinary spaces and printable Unicode remain intact — addressed by scenario 3.
- Request: the existing line-break case still passes (a line feed becomes one space) — addressed by scenario 4 and by the unchanged assertion in shape/discovery_test.go.
- Request scope: shape/documentation.go and shape/discovery_test.go only; lookup results, retained evidence formats and provider behavior are omitted from the change.

## Tests
- Every control character in a progress reference becomes a space: extend the progress assertions in shape/discovery_test.go with a reference holding CR, LF, tab, NUL and ESC and check the exact started line (unit)
- Ordinary spaces and printable Unicode stay intact: add a reference with a double space and non-ASCII text to shape/discovery_test.go and check it is unchanged (unit)
- The line-break case keeps producing a single space: the existing "missing\nprovider" assertion in shape/discovery_test.go stays unchanged (existing)
- End to end through the exported function: acceptance.feature scenarios run the Go driver (scenario)

## Feasibility
- none
