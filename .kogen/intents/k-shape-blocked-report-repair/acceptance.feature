Feature: Repairable feedback for a refused blocked report
  A blocked report is an exclusive, truthful report that a required edit touches a
  protected path. When it lists a path outside the protected set, the refusal tells
  the shaper that path is editable, shows the protected set the check used, and
  states the next move. These journeys run this module's real Shape command with a
  scripted shaper on a small synthetic repository.

  Scenario: Three unprotected paths are refused with repairable feedback, then files are accepted
    When k-shape-blocked-report-repair: the blocked report journey runs in "unprotected" mode
    Then k-shape-blocked-report-repair: the blocked report journey satisfies the contract

  Scenario: Only the unprotected paths of a mixed report are named, and the corrected report is accepted
    When k-shape-blocked-report-repair: the blocked report journey runs in "mixed" mode
    Then k-shape-blocked-report-repair: the blocked report journey satisfies the contract

  Scenario: A report of only protected paths with a reason is accepted and writes nothing
    When k-shape-blocked-report-repair: the blocked report journey runs in "protected" mode
    Then k-shape-blocked-report-repair: the blocked report journey satisfies the contract

  Scenario: Empty paths, a blank path and a non-text reason keep their existing problem lines
    When k-shape-blocked-report-repair: the blocked report journey runs in "malformed" mode
    Then k-shape-blocked-report-repair: the blocked report journey satisfies the contract

  Scenario: Without protect patterns the feedback lists the automatic baseline
    When k-shape-blocked-report-repair: the blocked report journey runs in "automatic" mode
    Then k-shape-blocked-report-repair: the blocked report journey satisfies the contract
