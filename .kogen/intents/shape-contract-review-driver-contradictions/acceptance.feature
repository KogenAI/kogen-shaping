Feature: The contract review also finds self-contradicting driver tests
  This feature covers the contract review of the shaper: its instructions name three kinds of
  self-contradiction inside a candidate step file, and a finding that quotes an earlier statement
  of the same step file is accepted as evidenced. A driver test of this Intent is added to package
  shape in a scratch copy of the checkout and run there.

  Scenario: The review instructions name the three self-contradiction categories
    Then shape-contract-review-driver-contradictions: the driver subtest "instructions name the removed or replaced record" passes
    And shape-contract-review-driver-contradictions: the driver subtest "instructions name the uniqueness collision" passes
    And shape-contract-review-driver-contradictions: the driver subtest "instructions name the request or tenant context" passes

  Scenario: The review instructions keep the existing exclusions
    Then shape-contract-review-driver-contradictions: the driver subtest "instructions keep the missing-feature exclusion" passes
    And shape-contract-review-driver-contradictions: the driver subtest "instructions keep the unsure rule" passes

  Scenario: A quote of an earlier statement of the same step file is evidence only when exact
    Then shape-contract-review-driver-contradictions: the driver subtest "earlier statement quote is evidenced" passes
    And shape-contract-review-driver-contradictions: the driver subtest "invented earlier statement quote is not evidenced" passes

  Scenario: A scripted review reporting such a finding refuses the draft
    Then shape-contract-review-driver-contradictions: the driver subtest "scripted review refuses the draft" passes
