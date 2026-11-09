Feature: The decision check treats the same value written in another notation as following the request
  Shape always checks a candidate's stated decisions against the request with one decision check
  request per checked submission. A decision that restates a request value in another notation
  follows the request and gets no feedback; contradicts reaches the shaper only with both values
  named and still different after normalising notation. Each case is a synthetic request and Intent
  pair shaped natively with a scripted shaper and a scripted decision-check model.

  Scenario Outline: A same-value restatement follows the request without a refusal or a second decision check
    When k-shape-decision-same-value: the "<case>" case is shaped
    Then k-shape-decision-same-value: exactly 1 decision check request is made
    And k-shape-decision-same-value: the decision check recorded "follows" for the first submission
    And k-shape-decision-same-value: no feedback to the shaper contains "Decision check:"
    And k-shape-decision-same-value: the Intent is published from the first submission

    Examples:
      | case              |
      | uploads-1000      |
      | retry-120-seconds |
      | display-fraction  |

  Scenario: The decision check compares the candidate's values with the request's after normalising notation
    When k-shape-decision-same-value: the "uploads-1000" case is shaped
    Then k-shape-decision-same-value: the decision check request carries the request and the candidate intent.md
    And k-shape-decision-same-value: the decision check request includes:
      """
      Compare values, not notation: before comparing, normalise digit grouping (1,000 and 1000), percentages and fractions (50% and 0.5), units of the same quantity (2 minutes and 120 seconds), date layouts, the order of a list whose order does not matter, and letter case where case does not matter.
      """
    And k-shape-decision-same-value: the decision check request includes:
      """
      A statement that restates the request's value in another notation follows the request.
      """
    And k-shape-decision-same-value: the decision check request includes:
      """
      Give contradicts only when the values still differ after that normalisation, such as a different number, a wrongly converted unit or a different order where order matters; then give intent_value (the value as the statement writes it), request_value (the value as a request quote writes it) and difference (what makes them differ). For follows and unsupported give these three as empty strings.
      """

  Scenario: A different number contradicts the request and the feedback names both values
    When k-shape-decision-same-value: the "uploads-10000" case is shaped
    Then k-shape-decision-same-value: the first submission is refused by the decision check with a line containing "verdict: contradicts", "intent_value: 10,000" and "request_value: 1,000"
    And k-shape-decision-same-value: the decision check recorded "contradicts" for the first submission
    And k-shape-decision-same-value: the Intent is published from the corrected submission

  Scenario: A different order where order matters contradicts the request
    When k-shape-decision-same-value: the "sort-oldest-first" case is shaped
    Then k-shape-decision-same-value: the first submission is refused by the decision check with a line containing "verdict: contradicts", "intent_value: oldest first" and "request_value: newest first"
    And k-shape-decision-same-value: the decision check recorded "contradicts" for the first submission
    And k-shape-decision-same-value: the Intent is published from the corrected submission

  Scenario: A scripted contradiction whose named values are the same value follows the request
    When k-shape-decision-same-value: the "uploads-1000-scripted-contradicts" case is shaped
    Then k-shape-decision-same-value: exactly 1 decision check request is made
    And k-shape-decision-same-value: the decision check recorded "follows" for the first submission
    And k-shape-decision-same-value: no feedback to the shaper contains "Decision check:"
    And k-shape-decision-same-value: the Intent is published from the first submission

  Scenario: A scripted contradiction that names no values is not presented as a contradiction
    When k-shape-decision-same-value: the "uploads-1000-contradicts-without-values" case is shaped
    Then k-shape-decision-same-value: the first submission is refused by the decision check with a line containing "verdict: unsupported", "statement: - Upload limit is 1000 rows (source: request)" and 'request_quotes: "limit uploads to 1,000 rows"'
    And k-shape-decision-same-value: the first submission's feedback does not contain "contradicts"
    And k-shape-decision-same-value: the decision check recorded "unsupported" for the first submission
    And k-shape-decision-same-value: the Intent is published from the corrected submission
