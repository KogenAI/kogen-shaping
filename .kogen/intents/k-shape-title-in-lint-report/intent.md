# Restore last-attempt title shortening

Slug: k-shape-title-in-lint-report
Model: not applicable — this repairs classification and title content transformation within the existing bounded retry workflow without adding workflow states
Acceptance:
- .kogen/intents/k-shape-title-in-lint-report/acceptance.feature
- features/steps/k-shape-title-in-lint-report__steps.js
- features/steps/k-shape-title-in-lint-report__fixture.js

Restore consumer-first final-attempt title shortening without reverting aggregate lint reporting or changing the coordinator pin. Recognize the title-length problem by stable identity, not complete message wording or position. Both inspected coordinator APIs provide field-prefixed Lines, not per-problem codes: use the `title:` field together with the actual overlong title. Keep shortenTitle and the retry limits unchanged.

On the final attempt, shorten even when another problem remains. Retain that shortened proposal, record title_shortened when shortening occurs, and recheck the complete candidate. Refuse remaining invalidity with current problems and never publish an invalid Intent; do not restore the original title. Earlier attempts still return full lint feedback without shortening. Preserve all non-title bytes. Update obsolete mixed-error tests and wording-pinned assertions to match this contract.

## Decisions
- Find the title problem anywhere in the full report by stable code or field without a diagnostic wording pin. (source: request)
- Keep shortened retained proposals while refusing remaining problems; never publish invalid content. (source: answer 1)
- Preserve final-attempt-only timing including retry limit 1 and the existing shortening algorithm. (source: request)

## Source comparison
- Addressed: K-SHAPE-TITLE-IN-LINT-REPORT reports that coordinator 8d24cc7 (Report every Intent lint problem in one refusal, 3 Nov) broke fallback; supplied delivery evidence against shaping main 8c5ea31 says 656694f passed and 8d24cc7 and later mains failed. These are historical supplied observations, not new measurements. The contract fixes the consumer, not aggregate reporting.
- Addressed: the reproductions TestSubmissionRetryOne/title_fallback (submission_retry_test.go:62, invalid_output after 1 attempts) and TestShapeShortensATooLongTitleOnlyOnTheLastAttempt/last_attempt (shape_test.go:1016, invalid_output after 3 attempts) are executed by acceptance against today's pin v0.0.0-20261102193107-c4a49a4fe84f and coordinator main resolved during shaping as v0.0.0-20261103214825-03d7341feab7, commit 03d7341feab73c5f5bfc79faec0e693427535f65.
- Superseded: existing another_problem_remains expects restoration and zero events. Answer 1 replaces that expectation with retained shortening and refusal. The supplied protected TestKTitleRetained fixture independently asserts retained title, non-title content, event count, refusal, and no publication; Build must also update the obsolete existing regression.
- Addressed: wording and ordering independence are exercised by local coordinator copies which change title diagnostics to `title: length exceeds allowed maximum` and place them before or after unrelated problems. Main itself and the pinned version remain unmodified in their respective scenarios. Existing feedback assertions must not pin the old titleTooLong message.
- Omitted: none of the requested behavior; no pin downgrade, provider change or new product setting is authorized.

## Tests
- Final-attempt fallback at retry limits 1 and 3, preserving earlier feedback without shortening: existing named reproductions on pinned and main dependencies (regression)
- Mixed errors retain shortened title and all other bytes but refuse publication: supplied TestKTitleRetained through actual Shape, plus update existing another_problem_remains (regression)
- Identity is independent of diagnostic wording and report order: run the same entry-point assertions against locally altered main reports with title first and last (integration)

## Feasibility
- A local probe resolved main and successfully ran existing TestShortenTitle (exit 0). Inspected downloaded project/intent.go for both versions: CheckIntent returns sys.Error Reason lint with field-prefixed Lines; main adds the measured length to the prior diagnostic. A source-inspection command exited 2 solely because coordinator does not own sys/error.go; this is not a CheckIntent rejection.
- Documentation search for the coordinator lint API returned generic GitHub results, not authoritative API documentation. No documentation success is claimed; downloaded module source establishes the used interface.
- evidence: {"id":"runner-observer","kind":"observer","inputs":["features/steps/k-shape-title-in-lint-report__steps.js","features/steps/k-shape-title-in-lint-report__fixture.js"],"observers":["features/steps/k-shape-title-in-lint-report__steps.js","features/steps/k-shape-title-in-lint-report__fixture.js"],"checks":[{"name":"good","command":"node features/steps/k-shape-title-in-lint-report__steps.js good","exit":0,"output":"accepted\n"},{"name":"bad","command":"node features/steps/k-shape-title-in-lint-report__steps.js bad","exit":1,"output":"rejected\n"}]}
