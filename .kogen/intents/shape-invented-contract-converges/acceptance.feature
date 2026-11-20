Feature: Invented literals converge
  This feature checks that Shape drops a repeated invented literal after one rewrite, keeps every other refusal, and that a Decision never weakens a guarantee.

  Scenario: A rewritten draft is approved without a record
    Given shape-invented-contract-converges: the harness has run
    Then shape-invented-contract-converges: case "rewritten" had first submission refused with "invented_contract", 0 dropped events and no card heading

  Scenario: A repeated literal is dropped and recorded
    Given shape-invented-contract-converges: the harness has run
    Then shape-invented-contract-converges: case "repeated" dropped 1 literals including "bound = 40" in the event and in the card

  Scenario: A new literal in the second submission is dropped
    Given shape-invented-contract-converges: the harness has run
    Then shape-invented-contract-converges: case "fresh" was refused once, dropped 1 literal "account_exports_url" and used 2 submissions

  Scenario: A dropped literal does not hide an uncovered failure mode
    Given shape-invented-contract-converges: the harness has run
    Then shape-invented-contract-converges: case "mixed" dropped "bound = 40" yet still refused "uncovered_failure_mode" on the second submission

  Scenario: The shaper instructions state the rule
    Given shape-invented-contract-converges: the harness has run
    Then shape-invented-contract-converges: shaper.md contains "is not asserted by the acceptance" and "never adds a mechanism" and "Request's own words"

  Scenario: A Decision never weakens a guarantee
    Given shape-invented-contract-converges: the harness has run
    Then shape-invented-contract-converges: the weakened Decision is refused with a fix line quoting "Deliver exactly once." and the full Decision is accepted
