Feature: The decision check refuses a Decision that weakens an absolute guarantee of the request
  This feature covers the decision check instructions of Shape. A Decision that restates an
  absolute guarantee such as "exactly once" as only its at-most-once, at-least-once or best-effort
  half is a contradiction even when it carries an assumption marker. A Decision that keeps the
  full guarantee is unaffected, and an unachievable guarantee is stated as an open question.

  Background:
    Given shape-exactly-once-reading-covers-interruption: the decision check harness has run

  Scenario: A weakened absolute guarantee is a contradiction even when assumption-marked
    Then shape-exactly-once-reading-covers-interruption: the scripted contradiction of the weakened decision is refused naming "at-most-once on a crash" and "exactly once"
    And shape-exactly-once-reading-covers-interruption: the decision instructions contain "at-most-once"
    And shape-exactly-once-reading-covers-interruption: the decision instructions contain "at-least-once"
    And shape-exactly-once-reading-covers-interruption: the decision instructions contain "best-effort"
    And shape-exactly-once-reading-covers-interruption: the decision instructions contain "even when the Decision carries an assumption marker"

  Scenario: A Decision that keeps the full guarantee is not affected
    Then shape-exactly-once-reading-covers-interruption: the scripted decision that keeps the full guarantee is not refused
    And shape-exactly-once-reading-covers-interruption: the decision instructions contain "keeps the full guarantee"

  Scenario: An unachievable guarantee is an open question and the rule is generic
    Then shape-exactly-once-reading-covers-interruption: the decision instructions do not contain "webhook"
    And shape-exactly-once-reading-covers-interruption: the decision instructions do not contain "delivery"
    And shape-exactly-once-reading-covers-interruption: the decision instructions do not contain "restart"
    And shape-exactly-once-reading-covers-interruption: the decision instructions contain "open question for the owner"
