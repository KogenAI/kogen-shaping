Feature: Overlap static and model checks
  This feature uses only the local scripted shape provider and a configured sleep marker.
  The one-second historical observations are not performance targets; these fixtures use two-second waits.

  Scenario: Overlap the two independent waits
    Given shape-static-checks-overlap-model-checks: the local scripted fixture is available
    When shape-static-checks-overlap-model-checks: one "passing" submission runs with two-second static and decision waits
    Then shape-static-checks-overlap-model-checks: the observed submission is
      """
      {
        "decision_calls": 1,
        "events": [
          "static_checks",
          "decision_check",
          "literal_check"
        ],
        "feedback": "Decision check: 1 stated decisions are not supported by the request; resolve each explicit conflict using the requirement below:\nstatement: - Non-integers raise TypeError. (source: request) | request_quotes: \"Non-integers raise TypeError.\" | verdict: contradicts | intent_value: TypeError | request_value: raise TypeError | difference: they differ | why: because\nfix for intent.md line 12: Replace \"TypeError\" with the request's requirement \"raise TypeError\". Align the acceptance expectations with that requirement.\n\nLiteral check: 1 required literal values are not carried exactly by the package; copy each value byte for byte into the Intent or its acceptance where the Intent uses it (descriptive text may stay paraphrased), or restore the substituted value:\nliteral: 4242 | source: request | source_quote: \"Use limit 4242.\" | status: missing | why: w",
        "gates": [
          "decision_check",
          "literal_check"
        ],
        "literal_calls": 1,
        "mode": "passing",
        "sleep_runs": 1,
        "totals": [
          1,
          1
        ],
        "min_ms": 1900,
        "max_ms": 3500
      }
      """

  Scenario: Discard model answers when shaped files fail static checks
    Given shape-static-checks-overlap-model-checks: the local scripted fixture is available
    When shape-static-checks-overlap-model-checks: one "refused" submission runs with two-second static and decision waits
    Then shape-static-checks-overlap-model-checks: the observed submission is
      """
      {
        "decision_calls": 1,
        "events": [
          "static_checks"
        ],
        "feedback": "Static checks: the shaped files fail 1 configured static check(s); nothing was published. Each check ran unchanged on the final submitted bytes in a fresh copy of the candidate; repair the named files so that every check passes. Resubmitting this package unchanged is refused again.\n- check slow: exit 1; files: features/steps/is-even__steps.js; command: sleep 2; echo done >> <marker>; ! grep -q BAD features/steps/is-even__steps.js\n  | (no output)",
        "gates": [
          "static_checks"
        ],
        "literal_calls": 1,
        "mode": "refused",
        "sleep_runs": 3,
        "totals": [
          1
        ],
        "min_ms": 1900,
        "max_ms": 15000
      }
      """
