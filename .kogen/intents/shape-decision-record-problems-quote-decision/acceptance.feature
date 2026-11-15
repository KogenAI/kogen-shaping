Feature: Decision record problems quote the decision they refuse
  This feature checks the problem lines of a refused sourced decision. Each case is a native Shape
  run with a scripted shaper whose first submission carries decisions the decision record gate
  refuses.

  Scenario: The refused second decision is named and quoted
    When shape-decision-record-problems-quote-decision: the "second-unsourced" case is shaped
    Then shape-decision-record-problems-quote-decision: the refusal problem lines are exactly:
      """
      - sourced decision 2 ("Round halves up"): no evidence and no uncertainty; cite what this shaping read or ran, or state the uncertainty you decided under
      """

  Scenario: A long decision is quoted to its first 120 characters
    When shape-decision-record-problems-quote-decision: the "long-text" case is shaped
    Then shape-decision-record-problems-quote-decision: the refusal problem lines are exactly:
      """
      - sourced decision 1 ("Use the documented rounding rule for every halfway value so that 2.5 and 3.5 are rounded in the same direction, and keep"): no evidence and no uncertainty; cite what this shaping read or ran, or state the uncertainty you decided under
      """

  Scenario: A decision of exactly 120 characters is quoted whole
    When shape-decision-record-problems-quote-decision: the "exact-120" case is shaped
    Then shape-decision-record-problems-quote-decision: the refusal problem lines are exactly:
      """
      - sourced decision 1 ("Use the documented rounding rule for every halfway value so that 2.5 and 3.5 are rounded in the same direction, and keep"): no evidence and no uncertainty; cite what this shaping read or ran, or state the uncertainty you decided under
      """

  Scenario: The quote shrinks so the whole line keeps its reason within 300 characters
    When shape-decision-record-problems-quote-decision: the "crowded-line" case is shaped
    Then shape-decision-record-problems-quote-decision: the only problem line has 300 characters, starts with "- sourced decision 1 (\"Use the documented rounding rule for eve" and ends with ".rb is not a file tracked at HEAD"
