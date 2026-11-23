Feature: A repeated guarantee conflict ends in one question to the caller
  This feature shapes a synthetic webhook request that requires exactly-once delivery over HTTP, with a scripted shaper and a scripted decision-check model.

  Scenario: A guarantee conflict reported twice ends with one question
    When shape-escalate-repeated-guarantee-conflict: the "webhook-twice" case is shaped
    Then shape-escalate-repeated-guarantee-conflict: shaping ends with questions and exit 5
    And shape-escalate-repeated-guarantee-conflict: exactly 2 decision check requests are made
    And shape-escalate-repeated-guarantee-conflict: exactly 2 shaper requests are made
    And shape-escalate-repeated-guarantee-conflict: the Intent is not published

  Scenario: The question offers retry or leave unresent and authorizes neither
    When shape-escalate-repeated-guarantee-conflict: the "webhook-twice" case is shaped
    Then shape-escalate-repeated-guarantee-conflict: the output offers an option that retries the unconfirmed delivery
    And shape-escalate-repeated-guarantee-conflict: the output offers an option that leaves the unconfirmed delivery unresent
    And shape-escalate-repeated-guarantee-conflict: no option is marked recommended
    And shape-escalate-repeated-guarantee-conflict: the output names the guarantee "exactly once"

  Scenario: A conflict reported once and then resolved is published
    When shape-escalate-repeated-guarantee-conflict: the "webhook-resolved" case is shaped
    Then shape-escalate-repeated-guarantee-conflict: the Intent is published after 2 decision check requests
    And shape-escalate-repeated-guarantee-conflict: the output asks no question
