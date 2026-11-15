Feature: Carried-decision problems show the lines they are about
  This feature runs the carried decisions gate on a ledger holding a superseded decision D1
  ("Keep the old message") and a standing decision D2 ("Use the new message"), and a draft whose
  Decisions section carries D1's line and a D2 line with changed text.

  Scenario: The carried-decision problems show the lines they are about
    When shape-carried-decisions-problems-show-lines: the gate checks the draft with a superseded and a changed decision
    Then shape-carried-decisions-problems-show-lines: the refusal problem lines are exactly:
      """
      - D1 is not a recorded decision that stands; remove its identity or submit a sourced decision: - Keep the old message (decision: D1, round 1) (source: request)
      - D2 does not match its recorded line; keep it as: - Use the new message (decision: D2, round 2) (source: request) (Intent has: - Use the newer message (decision: D2, round 2) (source: request))
      - D2 (round 2) is not in Decisions; keep it as: - Use the new message (decision: D2, round 2) (source: request)
      """
