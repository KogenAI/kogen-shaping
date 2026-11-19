Feature: A refusal with contract findings also carries the controls.json problems that need no run
  This feature covers the refusal of the contract review gate. When the review has confirmed
  findings and a counted observable property, the same refusal lists the controls.json problems
  that need no model call and no run: the file does not exist, or it fails the schema check. No
  implementation is run for a draft with a finding. A review without a counted observable
  property adds no controls line, and the refusal of a clean review is as before.

  Background:
    Given shape-missing-controls-reported-with-findings: the scripted shaping rounds have run

  Scenario: A finding and a counted property without controls.json list both in one refusal
    Then shape-missing-controls-reported-with-findings: the problems of "missing" start with:
      """
      ["- Even numbers | expectation: ", "- fix for Even numbers: in ", "- controls.json: .kogen/intents/is-even/controls.json does not exist; write it as {"]
      """
    And shape-missing-controls-reported-with-findings: the feedback of "missing" has 5 lines
    And shape-missing-controls-reported-with-findings: the feedback of "missing" starts with 'Contract check: 1 acceptance expectation(s)'

  Scenario: A finding and a controls.json that violates the schema list both
    Then shape-missing-controls-reported-with-findings: the problems of "schema" include the line:
      """
      - controls.json: wrong implementation "w": scenario "No such scenario" is not a scenario of acceptance.feature
      """
    And shape-missing-controls-reported-with-findings: the problems of "schema" start with the contract finding lines

  Scenario: A finding without a counted observable property gets no controls line
    Then shape-missing-controls-reported-with-findings: the problems of "uncounted" start with:
      """
      ["- Even numbers | expectation: ", "- fix for Even numbers: in "]
      """
    And shape-missing-controls-reported-with-findings: the feedback of "uncounted" has 3 lines

  Scenario: A clean review without controls.json is refused with the single line it gives today
    Then shape-missing-controls-reported-with-findings: the problems of "clean" start with:
      """
      ["- controls.json: .kogen/intents/is-even/controls.json does not exist; write it as {"]
      """
    And shape-missing-controls-reported-with-findings: the feedback of "clean" has 2 lines
    And shape-missing-controls-reported-with-findings: the feedback of "clean" starts with 'Wrong-implementation check: 1 problem(s) with whether acceptance rejects a wrong implementation'

  Scenario: A deferred or a valid controls.json adds nothing to a refusal with a finding
    Then shape-missing-controls-reported-with-findings: the problems of "deferred" start with:
      """
      ["- Even numbers | expectation: ", "- fix for Even numbers: in "]
      """
    And shape-missing-controls-reported-with-findings: the problems of "valid" start with:
      """
      ["- Even numbers | expectation: ", "- fix for Even numbers: in "]
      """

  Scenario: No implementation is run for a draft with a finding
    Then shape-missing-controls-reported-with-findings: the runs of "missing" are 0
    And shape-missing-controls-reported-with-findings: the runs of "schema" are 0
    And shape-missing-controls-reported-with-findings: the runs of "valid" are 0
    And shape-missing-controls-reported-with-findings: the runs of "deferred" are 0
