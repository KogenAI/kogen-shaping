# Tell the shaper when its title was shortened

Slug: shape-shortened-title-note-tells-shaper
Model: not applicable — this adds one submission-result note and changes no shaping state or transitions
Acceptance:
- .kogen/intents/shape-shortened-title-note-tells-shaper/acceptance.feature
- features/steps/shape-shortened-title-note-tells-shaper__steps.js
- features/steps/shape-shortened-title-note-tells-shaper__driver/run.js
- features/steps/shape-shortened-title-note-tells-shaper__driver/observe_test.go.txt

When Shape shortens a title longer than 50 characters, the shaper is never told, so it resubmits the long title and Kogen shortens it again every round: the line `kogen: shortened the title to 50 characters: Add an is-even check for integers and also reject` style progress output repeats after each `shaping (external round N)` line. When the title is shortened, Shape adds one note to the submission's notes: `the title was shortened to 50 characters: <title>; keep this title in later submissions`, with the shortened title. The text the shaper receives for that submission then starts with `note:` and that line, whether the submission goes on to pass or is refused at a later gate. No note is added when no title was shortened (a title of 50 characters or fewer, or a lint failure with other lines than the title length).

The progress line, the `title_shortened` event, the retained proposal and every gate result stay exactly as they are. No change to which drafts pass and no new configuration.

Implementation: one narrow change in shape/shape.go in the files-submission path where `title_shortened` is recorded and `sh.bareProgress("kogen: shortened the title to 50 characters: " + short)` is written; the note is appended to sh.notes there. The existing submit code `text = "note:\n" + strings.Join(sh.notes, "\n") + "\n" + text` already prefixes the notes to the submission result and stays unchanged. Add one focused model-free Go test in the shape package next to TestShapeShortensATooLongTitleAtOnce in shape/shape_test.go, using its helpers (shpTitled, shpSubmit, newShpFx, f.Prov.requests(), shpOutputs). The test covers: a 53-character title with a later-gate-refused decision line (the c1 output contains `note:` and the note line); a title-only lint failure that passes every other gate (one `title_shortened` event, one progress line `kogen: shortened the title to 50 characters: Add an is-even check for integers and also reject`); a 50-character title (no note text, no event).

## Decisions
- Add the note by appending one line to sh.notes where the title is shortened, so the existing `note:` prefix in submit carries it for passing and refused submissions alike. (decision: D1, round 1) (evidence: shape/shape.go) (source: request)
- The note text is exactly `the title was shortened to 50 characters: <shortened title>; keep this title in later submissions`; the progress line, event, retained proposal and gate results are untouched, and no setting is added. (decision: D2, round 1) (evidence: shape/shape.go; shape/shape_test.go) (source: request)
- Acceptance observes Shape through a supplied scripted Go observer using the existing shape test helpers; the refused case uses a decision whose evidence path is not tracked, which refuses at the decision_evidence gate after the title is shortened, and the 50-character case uses the same refusal so that its output text exists to inspect. (decision: D3, round 1) (evidence: shape/shape_test.go; shape/helper_test.go) (assumption: a passing submission ends shaping without a further request, so the note text is observable only through refused submissions) (source: request)
- No Quint model: the change is one note, not a state transition. (decision: D4, round 1) (evidence: shape/shape.go) (source: request)

## Source comparison
- Addressed: the request's expected note text, its placement via sh.notes and the `note:` prefix (`text = "note:\n" + strings.Join(sh.notes, "\n") + "\n" + text`) are the outcome paragraph and scenario 1; the unchanged progress line, event, retained proposal and gates are preserved by scenarios 2 and 3 and the implementation boundary.
- Addressed: the request's three scenarios map to the three feature scenarios, each with its own wrong-implementation control in controls.json (no note; progress line twice; note for every submission).
- Addressed: the request's constraints (no note without shortening, no change to which drafts pass, no new configuration) are the outcome paragraph and Decisions.
- Addressed: the observed AoR climb (release 891c91b3, the line `kogen: shortened the title to 50 characters: Deliver board webhooks once and switch off failing` after the `shaping (external round N)` lines in rounds 1, 2 and 3) is historical context for the problem; it is not reproduced from external lab files. The progress call `sh.bareProgress("kogen: shortened the title to 50 characters: " + short)` is preserved unchanged.
- Omitted: nothing in the request is deliberately omitted. Conflicting or superseded: none.

## Tests
- A shortened title is announced to the shaper in the submission output, including when a later gate refuses: add a subtest in shape/shape_test.go next to TestShapeShortensATooLongTitleAtOnce checking the c1 output contains `note:` and the note line, and compare the same through the real Shape observer in acceptance (regression)
- A title-only lint failure is shortened once with one event and one progress line: keep the existing title-only-lint subtest of TestShapeShortensATooLongTitleAtOnce (existing)
- A title of 50 characters gets no note and no event: add the no-note assertion to the fifty-characters subtest of that test and observe it in acceptance (regression)

## Feasibility
- The observer builds the supplied harness into the shape test binary with Go overlays and ran against the unchanged tree: a 53-character title with an untracked-evidence decision refuses at decision_evidence with the title already shortened, one request later; a passing submission makes no further request. (probes: exit 0, 2 s)
- evidence: {"id":"title-note-observer","kind":"observer","inputs":[".kogen/intents/shape-shortened-title-note-tells-shaper/acceptance.feature",".kogen/intents/shape-shortened-title-note-tells-shaper/controls.json","features/steps/shape-shortened-title-note-tells-shaper__driver/observe_test.go.txt","features/steps/shape-shortened-title-note-tells-shaper__driver/run.js"],"observers":["features/steps/shape-shortened-title-note-tells-shaper__driver/run.js","features/steps/shape-shortened-title-note-tells-shaper__driver/observe_test.go.txt"],"checks":[{"name":"good","command":"node features/steps/shape-shortened-title-note-tells-shaper__driver/run.js good","exit":0,"output":"observer accepted the correct implementation\n"},{"name":"bad","command":"node features/steps/shape-shortened-title-note-tells-shaper__driver/run.js bad","exit":1,"output":"observer rejected all three wrong implementations\n"}]}
