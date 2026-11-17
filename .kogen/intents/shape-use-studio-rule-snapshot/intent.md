# Use the Studio rule snapshot in Shape

Slug: shape-use-studio-rule-snapshot
Model: not applicable — the change loads and records a read-only snapshot; it has no meaningful state machine of its own
Acceptance:
- .kogen/intents/shape-use-studio-rule-snapshot/acceptance.feature
- features/steps/shape-use-studio-rule-snapshot__steps.js
- features/steps/shape-use-studio-rule-snapshot__harness.js

Shape consumes the shared scoped Studio rule snapshot that kogen-coordinator landed (package `rules`, commit 3b8df0c0: `rules.Load(ctx, root, rules.Options{Getenv, Now, Log, Client}) (*rules.Snapshot, error)`, configuration file `.kogen/studio-rules.yaml`, errors of class `rules/refused`, `rules/hold`, `rules/unavailable`, `rules.ErrNotConfigured` for a standalone project; reasons `token_missing`, `http_401`, `http_403`, `http_503`, `unreachable`, `malformed_response`). Step files of this Intent follow features/steps/shape-use-studio-rule-snapshot__<suffix>.js: here `__steps.js` and `__harness.js`.

At the real run entry (`shape.Shape`, which also serves the answer run after a resume) Shape calls `rules.Load` with the project root, reading the token variable named by the configuration from the session environment. The applicable rules (id, revision, scope, topic, text, owner, overrides) are put into the native request of the shaper and of the contract review and confirmation, and of the decision and literal checks. Every run loads again, so a changed revision reaches the next run with no prompt-file edit.

Receipt: in the shaping events file `.kogen/local/shaping/<slug>/events.ndjson` Shape appends, when rules are loaded, one JSON event per run `{"event":"studio_rules","boundary":"run"|"answer","origin":"live"|"cached","snapshot_hash":<Snapshot.ContentHash>,"rules":[{"id":<id>,"revision":<revision exactly as Studio sent it>}]}`; boundary is `answer` when the run's input answers pending questions. When the load is refused or unavailable, Shape stops before any model call with an error whose text contains the producer's `class: reason`, and appends `{"event":"studio_rules_refused","class":...,"reason":...}`. With no `.kogen/studio-rules.yaml` (`rules.ErrNotConfigured`) behavior is exactly as before and no such event is written. A cached snapshot is used only by the producer for an outage of exactly the same scope; a 401 or 403 refuses and never reuses a cache. No token appears in a request, log, event or source. Rule text is advisory and never overrides approval, checks or protected paths. The go.mod pin of kogen-coordinator advances to the version containing `rules` (go.mod and go.sum are not protected by the current contract). The Review-repository finding link is out of scope.

## Decisions
- Shape reuses the landed `rules` package of kogen-coordinator and adds no HTTP, cache or precedence code (evidence: go.mod) (source: request)
- The receipt is the `studio_rules` event of the shaping events file defined above, since Shape has no other run receipt (evidence: shape/evidence.go) (source: request)
- Rules are loaded at the start of every run and never held in process, so a changed revision reaches the next run (source: request)
- The producer source was read from the landed commit bundle, not a published module version; the Build confirms the exact pseudo-version and names against the pinned module (assumption: exact landed pseudo-version of kogen-coordinator commit 3b8df0c0) (source: request)

## Source comparison
- Rules in the shaper request and receipt: scenario 1. Changed revision: scenario 2. Standalone: scenario 3. Answer boundary: scenario 4. Outage cache: scenario 5. Revoked token http_401: scenario 6; foreign scope (http_403), other-scope outage and no-cache cases use the same refusal path and are covered by the journeys `foreign`, `foreign_down` and `cold_down` of the harness, which the Build must keep working.
- Review finding link: omitted, owned by the Review member.

## Tests
- Rules reach the requests and receipt: harness journey live (scenario)
- Changed revision reaches the next run: journey changed (scenario)
- Unconfigured project unchanged: journey standalone (scenario)
- Answer run records the rules: journey answer (scenario)
- Outage uses the same-scope cache: journey outage (scenario)
- Denied, foreign, other-scope outage and cold cache are named: journeys denied, foreign, foreign_down, cold_down (scenario)

## Feasibility
- none
