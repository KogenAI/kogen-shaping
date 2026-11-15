# Show near package text on missing literal lines

Slug: shape-literal-missing-shows-near-package-text
Model: not applicable — the change formats one feedback line from its inputs and keeps no state
Acceptance:
- .kogen/intents/shape-literal-missing-shows-near-package-text/acceptance.feature
- features/steps/shape-literal-missing-shows-near-package-text__steps.js

In the literal check (the `literal_check` event and its feedback), `literalProblemLine` (shape/literal.go) renders a `missing` item with `package_value: <the text in the package that the model reported as the closest match>` between `status: missing` and `why:` when the model item has a non-empty package_value, using the same field and position a `changed` item uses. The value is redacted like the other fields. A missing item with an empty or blank package_value renders exactly the line it renders today. The line of a `changed` item, the Literal check headline, the statuses the check assigns, and which submissions pass or are refused do not change. No setting is added.

Motivating case: a changed item shows `package_value: Free (1,000 cards, 1 GB)` and `package_value: USD 20/month` for the literal `$20/month`; on the next attempt the same literals were reported missing with only `why: Approved Free plan limits; the package carries Free (1,000 cards, 1 GB).`, so the near-miss text was visible only in the free-form why. With this change that text appears in the `package_value:` field of the missing line.

## Decisions
- A missing item shows package_value only when the model's value is non-empty after trimming whitespace, matching how `literalOutcome` treats blank values; otherwise the line is today's byte for byte. (evidence: shape/literal.go) (source: request)
- The field sits after `status: missing` and before `why:`, in the same format as the `changed` line, and is redacted with the same redactor. (evidence: shape/literal.go; shape/literal_test.go) (source: request)
- `literalOutcome` status rules, the headline and the event items are untouched; only `literalProblemLine` changes. (evidence: shape/literal.go) (source: request)
- The acceptance injects its own Go test through a go build overlay, so it does not depend on a test the Build must invent; the Build still adds the focused regression test to shape/literal_test.go. (evidence: shape/literal_test.go) (source: request)

## Source comparison
- Missing line carries `package_value: <the text in the package that the model reported as the closest match>` after status when the model gave one: addressed in the acceptance scenario "A missing literal with a near package value shows it after the status".
- Missing line without a package_value is today's line byte for byte: addressed in "A missing literal without a near package value keeps the old line".
- Changed line unchanged (`package_value: Free (1,000 cards, 1 GB)`, `package_value: USD 20/month` for `$20/month` are the observed changed form): addressed in "A changed literal renders as before".
- The observed missing form `why: Approved Free plan limits; the package carries Free (1,000 cards, 1 GB).` is the case the new field replaces as the only carrier of the near-miss text; value redacted like the others: addressed in "The near package value is redacted".
- Headline and status rules unchanged (`literal_check` event): addressed in "The headline and the statuses are unchanged".

## Tests
- A missing item whose model output has package_value renders `package_value:` after status: add a case to the "feedback, event and redaction" subtest of TestLiteralOutcomeRules in shape/literal_test.go (unit)
- A missing item without package_value renders the old line byte for byte: existing line assertions in shape/literal_test.go plus a blank-value case (unit)
- A changed item renders as before: existing changed line assertion in shape/literal_test.go (existing)
- The near value is redacted: a case with a secret in package_value in shape/literal_test.go (unit)

## Feasibility
- none
