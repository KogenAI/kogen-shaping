Feature: Contract-review refusals name the fix
  This feature runs Shape's real contract-review gate with a scripted review and a scripted
  confirmation in a copy of the shape tests' is-even project. When the confirmation confirms that an
  acceptance expectation contradicts the Intent, the refused item tells the shaper which exact
  expectation to change and the contract it has to follow, in a fix line of its own that follows the
  item line. The refusal fed back to the shaper, the saved not-ready details and the approval refusal
  keep the quotes and the fix. A corrected candidate passes, and a refuted or ungrounded finding
  refuses nothing.

  Scenario: A confirmed contradiction's refusal names the expectation to change and the contract to follow
    When shape-contract-refusal-fix-line: Shape shapes a package whose scripted contract review reports two confirmed contradictions and the shaper resubmits it unchanged
    Then shape-contract-refusal-fix-line: the refusal fed back after the first submission is:
      """
      Contract check: 2 acceptance expectation(s) contradict the Intent or code it keeps unchanged, so no correct implementation can pass them; nothing was published. Correct each expectation, or the Intent where the Intent is what is wrong, so that a correct implementation passes; do not delete or weaken a scenario to pass this check. Resubmitting this candidate unchanged is refused again.
      - Even numbers | expectation: .kogen/intents/is-even/acceptance.feature: Given is-even: the number 4 | contract: .kogen/intents/is-even/intent.md: calc.is_even(n) returns True for even integers. | why: four is even | confirmed: four is even
      - fix for Even numbers: in .kogen/intents/is-even/acceptance.feature change the expectation "Given is-even: the number 4" to follow the contract in .kogen/intents/is-even/intent.md: "calc.is_even(n) returns True for even integers."
      - Odd numbers | expectation: .kogen/intents/is-even/acceptance.feature: Given is-even: the number 3 | contract: .kogen/intents/is-even/intent.md: Non-integers raise TypeError. | why: three is an integer | confirmed: three is an integer
      - fix for Odd numbers: in .kogen/intents/is-even/acceptance.feature change the expectation "Given is-even: the number 3" to follow the contract in .kogen/intents/is-even/intent.md: "Non-integers raise TypeError."
      """
    And shape-contract-refusal-fix-line: the refusal fed back after the unchanged resubmission is the first refusal followed by:
      """
      Unchanged: this package is identical to the one refused at attempt 1; repair it before submitting again.
      """

  Scenario: The saved not-ready details keep the quotes and the fix of each confirmed finding
    When shape-contract-refusal-fix-line: Shape shapes a package whose scripted contract review reports two confirmed contradictions and the shaper resubmits it unchanged
    Then shape-contract-refusal-fix-line: the run exits 1
    And shape-contract-refusal-fix-line: the saved not-ready details contain these lines in order:
      """
      - Even numbers | expectation: .kogen/intents/is-even/acceptance.feature: Given is-even: the number 4 | contract: .kogen/intents/is-even/intent.md: calc.is_even(n) returns True for even integers. | why: four is even | confirmed: four is even
      - fix for Even numbers: in .kogen/intents/is-even/acceptance.feature change the expectation "Given is-even: the number 4" to follow the contract in .kogen/intents/is-even/intent.md: "calc.is_even(n) returns True for even integers."
      - Odd numbers | expectation: .kogen/intents/is-even/acceptance.feature: Given is-even: the number 3 | contract: .kogen/intents/is-even/intent.md: Non-integers raise TypeError. | why: three is an integer | confirmed: three is an integer
      - fix for Odd numbers: in .kogen/intents/is-even/acceptance.feature change the expectation "Given is-even: the number 3" to follow the contract in .kogen/intents/is-even/intent.md: "Non-integers raise TypeError."
      """

  Scenario: A corrected candidate passes the contract review
    When shape-contract-refusal-fix-line: Shape shapes a package whose scripted contract review reports one confirmed contradiction and the shaper then submits the corrected feature
    Then shape-contract-refusal-fix-line: the refusal fed back after the first submission ends with the line:
      """
      - fix for Even numbers: in .kogen/intents/is-even/acceptance.feature change the expectation "Given is-even: the number 4" to follow the contract in .kogen/intents/is-even/intent.md: "calc.is_even(n) returns True for even integers."
      """
    And shape-contract-refusal-fix-line: the run exits 0
    And shape-contract-refusal-fix-line: the contract_review events are:
      """
      [{"result":"refused","findings":1,"evidenced":1,"confirmed":1},{"result":"clean","findings":0,"evidenced":0,"confirmed":0}]
      """
    And shape-contract-refusal-fix-line: the Intent is published
    And shape-contract-refusal-fix-line: the corrected acceptance feature is saved
    And shape-contract-refusal-fix-line: the feedback after the corrected submission is not a contract refusal
    And shape-contract-refusal-fix-line: no contradictions record is saved

  Scenario: Refuted and ungrounded findings refuse nothing
    When shape-contract-refusal-fix-line: Shape shapes a package whose scripted contract review reports one refuted finding and two ungrounded findings
    Then shape-contract-refusal-fix-line: the run exits 0
    And shape-contract-refusal-fix-line: the contract_review events are:
      """
      [{"result":"clean","findings":3,"evidenced":1,"confirmed":0}]
      """
    And shape-contract-refusal-fix-line: the Intent is published
    And shape-contract-refusal-fix-line: no contract_review refusal event is saved
    And shape-contract-refusal-fix-line: no contradictions record is saved
    And shape-contract-refusal-fix-line: no feedback fed back to the shaper contains "fix for"

  Scenario: The refusal keeps its counts and its bounds with a long contract quote
    When shape-contract-refusal-fix-line: Shape shapes a package whose scripted contract review reports one confirmed contradiction whose contract quote is long
    Then shape-contract-refusal-fix-line: the contract_review events are:
      """
      [{"result":"refused","findings":1,"evidenced":1,"confirmed":1}]
      """
    And shape-contract-refusal-fix-line: the refusal fed back after the first submission starts with:
      """
      Contract check: 1 acceptance expectation(s) contradict the Intent or code it keeps unchanged,
      """
    And shape-contract-refusal-fix-line: the saved refusal event problems are the first 300 characters of the record lines, and the fix problem begins:
      """
      - fix for Even numbers: in .kogen/intents/is-even/acceptance.feature change the expectation "Given is-even: the number 4" to follow the contract in .kogen/intents/is-even/intent.md: "calc.is_even(n) returns True for even integers, which
      """
    And shape-contract-refusal-fix-line: the contradictions record keeps the full fix line, which ends with:
      """
      whatever its sign or size."
      """
