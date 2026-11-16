Feature: Report budget stops without a false no-submit claim
  This feature changes refusal text only. The observed AoR climb release 891c91b3
  cell /home/lanes/work.noindex/aor-climb/hypotheses/h20-main-head-71bee33c-vs-baseline-891c91b3/cells/rails-ft-saas-billing__baseline__r2
  stdout.txt said "the shaper used its 15-minute shaping budget (15 min 21 s elapsed, 3 requests) without a submission that passed; shaping stopped",
  then "the shaper ended without calling submit", despite attempt 1 through attempt 3
  being refused by decision_check, contract_review, contract_review. Its
  shaping-events.ndjson recorded three submission_refused events; h19 cells had
  the same misleading line. These are supplied historical observations, not local fixtures.
  The local reproduction uses two cheap refused submissions and the existing scripted clock.
  Only the false line is removed; detail, last refusal, earliest unresolved refusal,
  optional last tool-free answer, budget and ordered attempt lines are preserved.

  Scenario: Budget after two refused submissions
    Given shape-budget-stop-drops-no-submit-line: the supplied scripted Shape observer is available
    When shape-budget-stop-drops-no-submit-line: the scripted shapings stop
    Then shape-budget-stop-drops-no-submit-line: the "submitted" refusal equals
      """
      {"exit":1,"class":"shape","reason":"invalid_output","detail":"the shaper used its 15-minute shaping budget (16 min elapsed, 2 requests) without a submission that passed; shaping stopped","lines":["last refusal (attempt 2, gate validation):","second.go: shaping may write only files under .kogen/intents/is-even/ and step files under features/steps/","earliest unresolved refusal (attempt 1, gate validation):","first.go: shaping may write only files under .kogen/intents/is-even/ and step files under features/steps/","shaping budget: 900 s (15 min, wall-clock for the whole shaping, all rounds; the response_budget event is the separate limit on one provider response wait)","attempt 1: gate validation, model request 480 s, Kogen checks 0 s, 420 s remaining","attempt 2: gate validation, model request 480 s, Kogen checks 0 s, 0 s remaining"],"requests":2,"refusals":2}
      """

  Scenario: Budget without submission and no-tool-limit refusals retain the line
    Given shape-budget-stop-drops-no-submit-line: the supplied scripted Shape observer is available
    When shape-budget-stop-drops-no-submit-line: the scripted shapings stop
    Then shape-budget-stop-drops-no-submit-line: the "none" refusal equals
      """
      {"exit":1,"class":"shape","reason":"invalid_output","detail":"the shaper used its 15-minute shaping budget (16 min elapsed, 2 requests) without a submission that passed; shaping stopped","lines":["the shaper ended without calling submit","shaping budget: 900 s (15 min, wall-clock for the whole shaping, all rounds; the response_budget event is the separate limit on one provider response wait)"],"requests":2,"refusals":0}
      """

    Then shape-budget-stop-drops-no-submit-line: the "no-tool" refusal equals
      """
      {"exit":1,"class":"shape","reason":"invalid_output","detail":"the shaper answered without calling any tool 3 consecutive times (no-tool limit); its last submission (attempt 2) was refused at gate validation","lines":["the shaper ended without calling submit","last refusal (attempt 2, gate validation):","second.go: shaping may write only files under .kogen/intents/is-even/ and step files under features/steps/","earliest unresolved refusal (attempt 1, gate validation):","first.go: shaping may write only files under .kogen/intents/is-even/ and step files under features/steps/","last tool-free answer: three"],"requests":5,"refusals":2}
      """

  Scenario: Budget after a tool-free answer preserves its text
    Given shape-budget-stop-drops-no-submit-line: the supplied scripted Shape observer is available
    When shape-budget-stop-drops-no-submit-line: the scripted shapings stop
    Then shape-budget-stop-drops-no-submit-line: the "answer" refusal equals
      """
      {"exit":1,"class":"shape","reason":"invalid_output","detail":"the shaper used its 15-minute shaping budget (18 min elapsed, 3 requests) without a submission that passed; shaping stopped; last tool-free answer: Still investigating.","lines":["last refusal (attempt 2, gate validation):","second.go: shaping may write only files under .kogen/intents/is-even/ and step files under features/steps/","earliest unresolved refusal (attempt 1, gate validation):","first.go: shaping may write only files under .kogen/intents/is-even/ and step files under features/steps/","last tool-free answer: Still investigating.","shaping budget: 900 s (15 min, wall-clock for the whole shaping, all rounds; the response_budget event is the separate limit on one provider response wait)","attempt 1: gate validation, model request 360 s, Kogen checks 0 s, 540 s remaining","attempt 2: gate validation, model request 360 s, Kogen checks 0 s, 180 s remaining"],"requests":3,"refusals":2}
      """
