Feature: Identify unchanged refused decision statements
  This feature reproduces the AoR climb observation from releases 4fb52c19 to
  30cf6619: 12 of 78 decision_check refusals repeated a statement, all but one
  on rails-ft-webhook-delivery-trust-and-restart, requirement "Deliver exactly once."
  Historical cells include h7-sonnet-review-vs-baseline-30cf6619 /
  rails-ft-webhook-delivery-trust-and-restart__baseline (1 to 2 and 2 to 3),
  h8-shape-opus-medium-vs-baseline-30cf6619 /
  rails-ft-webhook-delivery-trust-and-restart__baseline__r3 (1 to 2 and 2 to 3),
  and h10-build-opus-medium-vs-baseline-30cf6619 /
  rails-ft-webhook-delivery-trust-and-restart__baseline__r2 (1 to 2).
  These local scripted verdicts reproduce the unchanged statement and moving
  intent line without claiming to replay private shaping-events.ndjson.

  Scenario: Repeated refusal through real submissions
    Given shape-repeated-decision-statement-says-so: the scripted local decision submissions
      """
      {"mode": "submit", "request": "Deliver exactly once.", "rounds": [{"statement": "- Deliver at least once. (source: request)", "submission": 1, "verdict": "contradicts", "intentValue": "at least once", "requestValue": "exactly once"}, {"statement": "- Deliver at least once. (source: request)", "submission": 2, "verdict": "contradicts", "intentValue": "at least once", "requestValue": "exactly once"}], "why": "The delivery guarantee differs.", "difference": "different delivery guarantees"}
      """
    When shape-repeated-decision-statement-says-so: the decision submissions are checked
    Then shape-repeated-decision-statement-says-so: only the expected additive notes appear
      """
      ["", "note: this statement is unchanged since submission 1 and was refused for the same reason; change the statement itself, not the text around it"]
      """

  Scenario: Changed statement through real submissions
    Given shape-repeated-decision-statement-says-so: the scripted local decision submissions
      """
      {"mode": "submit", "request": "Deliver exactly once.", "rounds": [{"statement": "- Deliver at least once. (source: request)", "submission": 1, "verdict": "contradicts", "intentValue": "at least once", "requestValue": "exactly once"}, {"statement": "- Deliver at most once. (source: request)", "submission": 2, "verdict": "contradicts", "intentValue": "at most once", "requestValue": "exactly once"}], "why": "The delivery guarantee differs.", "difference": "different delivery guarantees"}
      """
    When shape-repeated-decision-statement-says-so: the decision submissions are checked
    Then shape-repeated-decision-statement-says-so: only the expected additive notes appear
      """
      ["", ""]
      """

  Scenario: Whitespace normalization and actual submission numbers
    Given shape-repeated-decision-statement-says-so: the scripted local decision submissions
      """
      {"mode": "gate", "request": "Deliver exactly once.", "rounds": [{"statement": "- Deliver at least once. (source: request)", "submission": 2, "verdict": "contradicts", "intentValue": "at least once", "requestValue": "exactly once"}, {"statement": "- Deliver\t  at least once. (source: request)", "submission": 4, "verdict": "contradicts", "intentValue": "at least once", "requestValue": "exactly once"}], "why": "The delivery guarantee differs.", "difference": "different delivery guarantees"}
      """
    When shape-repeated-decision-statement-says-so: the decision submissions are checked
    Then shape-repeated-decision-statement-says-so: only the expected additive notes appear
      """
      ["", "note: this statement is unchanged since submission 2 and was refused for the same reason; change the statement itself, not the text around it"]
      """

  Scenario: Earlier nonadjacent refusal
    Given shape-repeated-decision-statement-says-so: the scripted local decision submissions
      """
      {"mode": "gate", "request": "Deliver exactly once.", "rounds": [{"statement": "- Deliver at least once. (source: request)", "submission": 1, "verdict": "contradicts", "intentValue": "at least once", "requestValue": "exactly once"}, {"statement": "- Deliver at most once. (source: request)", "submission": 2, "verdict": "contradicts", "intentValue": "at most once", "requestValue": "exactly once"}, {"statement": "- Deliver at least once. (source: request)", "submission": 3, "verdict": "contradicts", "intentValue": "at least once", "requestValue": "exactly once"}], "why": "The delivery guarantee differs.", "difference": "different delivery guarantees"}
      """
    When shape-repeated-decision-statement-says-so: the decision submissions are checked
    Then shape-repeated-decision-statement-says-so: only the expected additive notes appear
      """
      ["", "", "note: this statement is unchanged since submission 1 and was refused for the same reason; change the statement itself, not the text around it"]
      """

  Scenario: Passing verdict is never changed
    Given shape-repeated-decision-statement-says-so: the scripted local decision submissions
      """
      {"mode": "gate", "request": "Deliver exactly once.", "rounds": [{"statement": "- Deliver at least once. (source: request)", "submission": 1, "verdict": "contradicts", "intentValue": "at least once", "requestValue": "exactly once"}, {"statement": "- Deliver at least once. (source: request)", "submission": 2, "verdict": "follows", "intentValue": "at least once", "requestValue": "exactly once"}], "why": "The delivery guarantee differs.", "difference": "different delivery guarantees"}
      """
    When shape-repeated-decision-statement-says-so: the decision submissions are checked
    Then shape-repeated-decision-statement-says-so: only the expected additive notes appear
      """
      ["", ""]
      """

  Scenario: First refusal after an accepted statement
    Given shape-repeated-decision-statement-says-so: the scripted local decision submissions
      """
      {"mode": "gate", "request": "Deliver exactly once.", "rounds": [{"statement": "- Deliver at least once. (source: request)", "submission": 1, "verdict": "follows", "intentValue": "at least once", "requestValue": "exactly once"}, {"statement": "- Deliver at least once. (source: request)", "submission": 2, "verdict": "contradicts", "intentValue": "at least once", "requestValue": "exactly once"}], "why": "The delivery guarantee differs.", "difference": "different delivery guarantees"}
      """
    When shape-repeated-decision-statement-says-so: the decision submissions are checked
    Then shape-repeated-decision-statement-says-so: only the expected additive notes appear
      """
      ["", ""]
      """

