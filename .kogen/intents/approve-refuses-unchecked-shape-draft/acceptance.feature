Feature: approve-refuses-unchecked-shape-draft: Approval requires Shape's passed checks
  This feature runs Shape and Approve through scripted-provider journeys.

  Scenario: An interrupted Shape leaves a not-ready draft
    Then approve-refuses-unchecked-shape-draft: an interrupted Shape leaves a not-ready draft that approve refuses and Shape retries

  Scenario: A draft edited after its checks passed is not approvable
    Then approve-refuses-unchecked-shape-draft: a draft edited after its checks passed is not approvable

  Scenario: Checks recorded for a different request do not count
    Then approve-refuses-unchecked-shape-draft: checks recorded for a different request do not count

  Scenario: A draft that Shape took to ready keeps its normal behaviour
    Then approve-refuses-unchecked-shape-draft: a ready draft keeps its card, approval and exists answer
