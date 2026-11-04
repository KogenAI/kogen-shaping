Feature: Refuse acceptance that passes a wrong implementation
  Before Shape publishes an Intent whose request asks for properties a step can observe, Kogen
  runs the candidate's own acceptance in isolated copies with the implementations of the
  candidate's controls.json in place. With the correct implementation every scenario of those
  properties must pass; for each property, a wrong implementation must fail one of its scenarios
  at an assertion about it. Runs are capped per candidate, and plain requests run nothing more.

  Scenario: A bound that a slowed f still passes is refused, a measured bound publishes
    When k-shape-wrong-implementation: the "bound" case is shaped
    Then k-shape-wrong-implementation: candidate "only checks that f returns" is refused with a line naming "f is no slower than a same-run baseline plus 100 ms" and 'wrong implementation "slowed f" passes scenario "F stays within the bound"'
    And k-shape-wrong-implementation: candidate "only checks that f returns" made 3 shaper requests and 1 contract check request
    And k-shape-wrong-implementation: candidate "measures the bound" publishes
    And k-shape-wrong-implementation: the result of submission "cand" of candidate "measures the bound" does not contain "Wrong-implementation check"
    And k-shape-wrong-implementation: candidate "measures the bound" made 2 shaper requests and 1 contract check request
    And k-shape-wrong-implementation: candidate "measures the bound" ran acceptance 4 times

  Scenario: Kept behaviours A and B and new C each need a scenario that fails when they are removed
    When k-shape-wrong-implementation: the "kept" case is shaped
    Then k-shape-wrong-implementation: candidate "add step passes whatever calc returns" is refused with a line naming "add keeps working" and 'wrong implementation "add removed" passes scenario "Add keeps working"'
    And k-shape-wrong-implementation: the result of submission "cand" of candidate "add step passes whatever calc returns" does not contain "sub keeps working"
    And k-shape-wrong-implementation: the result of submission "cand" of candidate "add step passes whatever calc returns" does not contain "mul returns the product"
    And k-shape-wrong-implementation: candidate "fails when add, sub or mul is removed" publishes
    And k-shape-wrong-implementation: the result of submission "cand" of candidate "fails when add, sub or mul is removed" does not contain "Wrong-implementation check"
    And k-shape-wrong-implementation: candidate "fails when add, sub or mul is removed" ran acceptance 6 times
    And k-shape-wrong-implementation: the result of submission "cand" of candidate "exact, then loosened" does not contain "Wrong-implementation check"
    And k-shape-wrong-implementation: submission "change" of candidate "exact, then loosened" is refused with a line naming "add keeps working" and 'wrong implementation "add removed" passes scenario "Add keeps working"'

  Scenario: A step that fails for every implementation has no reachable pass
    When k-shape-wrong-implementation: the "unreachable" case is shaped
    Then k-shape-wrong-implementation: candidate "fails for every implementation" is refused with a line naming "f is no slower than a same-run baseline plus 100 ms" and "no pass is reachable"
    And k-shape-wrong-implementation: the result of submission "cand" of candidate "fails for every implementation" contains 'with the correct implementation in place, scenario "F stays within the bound" fails at "And is-even: f of 1000 returns both 2000 and 2001"'
    And k-shape-wrong-implementation: candidate "fails for every implementation" ran acceptance 3 times
    And k-shape-wrong-implementation: candidate "correct implementation disables assertions" is refused with a line naming "f is no slower than a same-run baseline plus 100 ms" and "no pass is reachable"

  Scenario: Running out of the run cap or the time cap refuses the candidate naming that cap
    When k-shape-wrong-implementation: the "caps" case is shaped
    Then k-shape-wrong-implementation: candidate "needs more runs than the cap" is refused with a line naming "run cap" and "above the cap of 8 runs per candidate"
    And k-shape-wrong-implementation: the result of submission "cand" of candidate "needs more runs than the cap" does not contain "time cap"
    And k-shape-wrong-implementation: candidate "needs more runs than the cap" ran acceptance 2 times
    And k-shape-wrong-implementation: candidate "runs longer than the time cap" is refused with a line naming "time cap" and "the total time cap of 2 seconds per candidate"
    And k-shape-wrong-implementation: the result of submission "cand" of candidate "runs longer than the time cap" does not contain "run cap"
    And k-shape-wrong-implementation: the result of submission "cand" of candidate "runs longer than the time cap" does not contain "no pass is reachable"
    And k-shape-wrong-implementation: candidate "runs longer than the time cap" ran acceptance 3 times

  Scenario: A plain request publishes as before with no extra runs and the same model requests
    When k-shape-wrong-implementation: the "plain" case is shaped
    Then k-shape-wrong-implementation: candidate "one command, one output" publishes
    And k-shape-wrong-implementation: candidate "one command, one output" made 2 shaper requests and 1 contract check request
    And k-shape-wrong-implementation: candidate "one command, one output" ran acceptance 2 times
    And k-shape-wrong-implementation: the result of submission "cand" of candidate "one command, one output" contains "Scenario: Hello prints hello"
    And k-shape-wrong-implementation: the result of submission "cand" of candidate "one command, one output" does not contain "Wrong-implementation check"

  Scenario: Failures that are not an assertion about the property are not evidence
    When k-shape-wrong-implementation: the "evidence" case is shaped
    Then k-shape-wrong-implementation: candidate "wrong add throws its own AssertionError" is refused with a line naming "add keeps working" and "with an assertion thrown outside the Intent's step files"
    And k-shape-wrong-implementation: candidate "wrong add fails an earlier unrelated step" is refused with a line naming "add keeps working" and 'at "Then is-even: calc takes 3 arguments", not at "And is-even: add of 2 and 3 gives an answer"'
    And k-shape-wrong-implementation: candidate "wrong add overwrites an assertion" is refused with a line naming "add keeps working" and 'wrong implementation "add removed" passes scenario "Add keeps working"'

  Scenario: Existing checks and Refine keep working as before
    When k-shape-wrong-implementation: the "existing" case is shaped
    Then k-shape-wrong-implementation: candidate "tests only add and mul" is refused by an existing check with a line naming "sub keeps working" and "no scenario exercises it"
    And k-shape-wrong-implementation: the result of submission "cand" of candidate "tests only add and mul" contains "Property check: "
    And k-shape-wrong-implementation: candidate "tests only add and mul" ran acceptance 2 times
    And k-shape-wrong-implementation: candidate "a step no implementation can pass" is refused by an existing check with a line naming "Then is-even: mul is fixed at 7" and "no pass is reachable"
    And k-shape-wrong-implementation: the result of submission "cand" of candidate "a step no implementation can pass" contains "Acceptance has steps that no implementation can pass"
    And k-shape-wrong-implementation: candidate "a step no implementation can pass" ran acceptance 4 times
    And k-shape-wrong-implementation: candidate "refined after publishing" publishes
    And k-shape-wrong-implementation: the refinement of candidate "refined after publishing" is ready and written
    And k-shape-wrong-implementation: the refinement of candidate "refined after publishing" ran acceptance 1 time
