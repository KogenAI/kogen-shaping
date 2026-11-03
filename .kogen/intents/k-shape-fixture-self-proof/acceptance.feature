Feature: Acceptance preparation succeeds on the unchanged tree before publication
  A scripted shaper submits the request's synthetic packages through the real Shape entry point,
  in a small repository whose acceptance runs the real Node and Cucumber in readiness's fresh copy.
  Every contract review answers no finding. A failed Given step, or a When or Then step that fails
  exactly as a refused preparation did, is refused at preparation; legitimate red of the behaviour
  under test keeps the ordinary acknowledgment route.

  Background:
    Given k-shape-fixture-self-proof: the preparation refusal begins with this line:
      """
      Acceptance preparation failed on the unchanged tree: 1 scenario(s) fail before they reach the behaviour under test, so this is not missing-feature red; nothing was published. A failed Given or Background step or Before hook, or a When or Then step that fails exactly as a refused preparation did, is a defect of the acceptance package: repair the preparation so that it succeeds on the unchanged tree and the consumer of its fixture accepts it. Keep the scenario and its outcome assertions, and do not change production code to accept a malformed fixture. Resubmitting or acknowledging this package unchanged is refused again.
      """

  Scenario: A Given whose fixture its consumer rejects is refused although contract review finds nothing
    When k-shape-fixture-self-proof: the scripted shaper runs the "missing-kind" journey
    Then k-shape-fixture-self-proof: submission 1 is refused at preparation
    And k-shape-fixture-self-proof: the answer to submission 1 has these lines in order:
      """
      - Count a prepared packet (acceptance.feature line 2): acceptance.feature line 3: Given fixture-proof: a packet failed (a Given step); acceptance exit 1
        AssertionError [ERR_ASSERTION]: fixturecheck exit=2
        missing kind=packet
        not run: acceptance.feature line 4: When fixture-proof: I count its items
        not run: acceptance.feature line 5: Then fixture-proof: the count is 2
      """
    And k-shape-fixture-self-proof: the refused submissions and their gates are "1:preparation"
    And k-shape-fixture-self-proof: nothing is published

  Scenario: An unchanged acknowledgment is refused again and repairing only the packet leaves legitimate red
    When k-shape-fixture-self-proof: the scripted shaper runs the "repair" journey
    Then k-shape-fixture-self-proof: submission 1 is refused at preparation
    And k-shape-fixture-self-proof: submission 2 is refused at preparation
    And k-shape-fixture-self-proof: the answer to submission 2 ends with "Unchanged: this package is identical to the one refused at attempt 1; repair it before submitting again."
    And k-shape-fixture-self-proof: submission 3 returns accounted red observations
    And k-shape-fixture-self-proof: the answer to submission 3 has these lines in order:
      """
      Scenario: Count a prepared packet (acceptance.feature line 2)
      acceptance.feature line 5: Then fixture-proof: the count is 2
      0 !== 2
      """
    And k-shape-fixture-self-proof: the refused submissions and their gates are "1:preparation 2:preparation"
    And k-shape-fixture-self-proof: the Intent is published
    And k-shape-fixture-self-proof: the readiness record keeps scenario "Count a prepared packet" failed

  Scenario: A fixture that breaks another rule of its consumer is refused with that rule's message
    When k-shape-fixture-self-proof: the scripted shaper runs the "items-not-array" journey
    Then k-shape-fixture-self-proof: submission 1 is refused at preparation
    And k-shape-fixture-self-proof: the answer to submission 1 has these lines in order:
      """
      - Count a prepared packet (acceptance.feature line 2): acceptance.feature line 3: Given fixture-proof: a packet failed (a Given step); acceptance exit 1
        AssertionError [ERR_ASSERTION]: fixturecheck exit=2
        items must be an array
      """
    And k-shape-fixture-self-proof: the answer to submission 1 does not contain "missing kind=packet"
    And k-shape-fixture-self-proof: nothing is published

  Scenario: A deliberately invalid input whose construction succeeds is not a preparation failure
    When k-shape-fixture-self-proof: the scripted shaper runs the "negative-input" journey
    Then k-shape-fixture-self-proof: the refused submissions and their gates are "none"
    And k-shape-fixture-self-proof: the Intent is published
    And k-shape-fixture-self-proof: the readiness record keeps scenario "A packet whose items are not an array is rejected" passed

  Scenario: The same failing builder moved into the When body is still refused at preparation
    When k-shape-fixture-self-proof: the scripted shaper runs the "moved-builder" journey
    Then k-shape-fixture-self-proof: submission 1 is refused at preparation
    And k-shape-fixture-self-proof: submission 2 is refused at preparation
    And k-shape-fixture-self-proof: the answer to submission 2 has these lines in order:
      """
      - Count a prepared packet (acceptance.feature line 2): acceptance.feature line 4: When fixture-proof: I count its items failed (the same failure as the preparation refused at attempt 1); acceptance exit 1
        AssertionError [ERR_ASSERTION]: fixturecheck exit=2
        missing kind=packet
        not run: acceptance.feature line 5: Then fixture-proof: the count is 2
      """
    And k-shape-fixture-self-proof: the refused submissions and their gates are "1:preparation 2:preparation"
    And k-shape-fixture-self-proof: nothing is published

  Scenario: A builder skipped behind a legitimate outcome failure stays unproven and unexecuted
    When k-shape-fixture-self-proof: the scripted shaper runs the "skipped-builder" journey
    Then k-shape-fixture-self-proof: submission 1 returns accounted red observations
    And k-shape-fixture-self-proof: the answer to submission 1 has these lines in order:
      """
      Unexercised acceptance steps: 1
      Skipped steps are unexercised; readiness does not prove their behavior or fixture validity.
      acceptance.feature line 5: Then fixture-proof: the count is 2
      0 !== 2
      acceptance.feature line 6: And fixture-proof: a second packet without its kind
      """
    And k-shape-fixture-self-proof: the refused submissions and their gates are "none"
    And k-shape-fixture-self-proof: no skipped step body ran
    And k-shape-fixture-self-proof: the Intent is published
    And k-shape-fixture-self-proof: the readiness record keeps scenario "Count a prepared packet" failed
    And k-shape-fixture-self-proof: the readiness record lists these unexercised steps of scenario "Count a prepared packet":
      """
      acceptance.feature line 6: And fixture-proof: a second packet without its kind
      """
