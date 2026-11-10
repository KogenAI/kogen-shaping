Feature: Count gate progress and give request-respecting repair feedback
  This feature exercises the small retry budget and decisionOutcome directly. Go regression tests cover conversation wiring.

  Scenario: Progress
    Given shape-retry-progress-policy: a fresh retry budget with limit three
    When shape-retry-progress-policy: refusals arrive as "decision_record,carried_decisions,decision_check"
    Then shape-retry-progress-policy: the budget trace is "1:1:false,1:2:false,1:3:false"

  Scenario: Same gate
    Given shape-retry-progress-policy: a fresh retry budget with limit three
    When shape-retry-progress-policy: refusals arrive as "decision_record,decision_record,decision_record"
    Then shape-retry-progress-policy: the budget trace is "1:1:false,2:2:false,3:3:true"

  Scenario: Total cap
    Given shape-retry-progress-policy: a fresh retry budget with limit three
    When shape-retry-progress-policy: refusals arrive as "decision_record,carried_decisions,decision_record,carried_decisions,decision_record,carried_decisions"
    Then shape-retry-progress-policy: the budget trace is "1:1:false,1:2:false,1:3:false,1:4:false,1:5:false,1:6:true"

  Scenario: Quoted unsupported
    Given shape-retry-progress-policy: statement "Report the initialized directory." in draft "heading\nReport the initialized directory."
    When shape-retry-progress-policy: an unsupported finding cites "Initialized Kogen in <path>"
    Then shape-retry-progress-policy: feedback contains "fix for intent.md line 2: the request speaks to this in the quoted passages; make the statement follow them or remove it; an assumption does not override request text." and assumption rule is "absent"

  Scenario: Unlocated quoted unsupported
    Given shape-retry-progress-policy: statement "Report the initialized directory." in draft "heading"
    When shape-retry-progress-policy: an unsupported finding cites "Initialized Kogen in <path>"
    Then shape-retry-progress-policy: feedback contains "fix for the quoted statement: the request speaks to this in the quoted passages; make the statement follow them or remove it; an assumption does not override request text." and assumption rule is "absent"

  Scenario: Silent unsupported
    Given shape-retry-progress-policy: statement "Report the initialized directory." in draft "heading\nReport the initialized directory."
    When shape-retry-progress-policy: an unsupported finding cites ""
    Then shape-retry-progress-policy: feedback contains "verdict: unsupported" and assumption rule is "present"
