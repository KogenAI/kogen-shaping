Feature: Blocked kept steps share one refusal line
  This feature checks the refusals that Shape raises for kept steps that never run on the unchanged tree, given two unchanged-tree observations of a scenario.

  Scenario: Three blocked kept steps of one property give one refusal
    Given shape-kept-blocked-lines-one-refusal: the property "P" names the kept steps on lines "3,4,5" of the scenario "S"
    And shape-kept-blocked-lines-one-refusal: the first observation shows the scenario "S" failed with "AssertionError: protected output lacks repaired text", with steps on lines "2,3,4,5" of which those on lines "3,4,5" never ran
    And shape-kept-blocked-lines-one-refusal: the second observation shows the scenario "S" failed with "AssertionError: protected output lacks repaired text", with steps on lines "2,3,4,5" of which those on lines "3,4,5" never ran
    When shape-kept-blocked-lines-one-refusal: the kept baseline refusals are computed
    Then shape-kept-blocked-lines-one-refusal: there are exactly 1 refusal
    And shape-kept-blocked-lines-one-refusal: the only refusal contains "lines 3, 4 and 5 is meant to assert unchanged behaviour but never runs on the unchanged tree"
    And shape-kept-blocked-lines-one-refusal: the only refusal contains "AssertionError: protected output lacks repaired text" 1 time

  Scenario: Two blocked kept steps are listed in ascending order and a step that ran is left out
    Given shape-kept-blocked-lines-one-refusal: the property "P" names the kept steps on lines "5,4,6" of the scenario "S"
    And shape-kept-blocked-lines-one-refusal: the first observation shows the scenario "S" failed with "AssertionError: protected output lacks repaired text", with steps on lines "3,4,5,6" of which those on lines "4,5,6" never ran
    And shape-kept-blocked-lines-one-refusal: the second observation shows the scenario "S" failed with "AssertionError: protected output lacks repaired text", with steps on lines "3,4,5,6,7" of which those on lines "4,5" never ran
    When shape-kept-blocked-lines-one-refusal: the kept baseline refusals are computed
    Then shape-kept-blocked-lines-one-refusal: there are exactly 1 refusal
    And shape-kept-blocked-lines-one-refusal: the only refusal contains "lines 4 and 5 is meant to assert unchanged behaviour"
    And shape-kept-blocked-lines-one-refusal: the only refusal does not contain the word "6"

  Scenario: A single blocked kept step keeps the sentence with line
    Given shape-kept-blocked-lines-one-refusal: the property "P" names the kept steps on lines "5" of the scenario "S"
    And shape-kept-blocked-lines-one-refusal: the first observation shows the scenario "S" failed with "AssertionError: protected output lacks repaired text", with steps on lines "3,4,5" of which those on lines "5" never ran
    And shape-kept-blocked-lines-one-refusal: the second observation shows the scenario "S" failed with "AssertionError: protected output lacks repaired text", with steps on lines "3,4,5" of which those on lines "5" never ran
    When shape-kept-blocked-lines-one-refusal: the kept baseline refusals are computed
    Then shape-kept-blocked-lines-one-refusal: the only refusal contains "scenario \"S\" line 5 is meant to assert unchanged behaviour but never runs on the unchanged tree"
    And shape-kept-blocked-lines-one-refusal: the only refusal does not contain the word "lines"

  Scenario: A red kept step alone keeps its own line
    Given shape-kept-blocked-lines-one-refusal: the property "P" names the kept steps on lines "4" of the scenario "S"
    And shape-kept-blocked-lines-one-refusal: the first observation shows the scenario "S" failed with "AssertionError: protected output lacks repaired text", with steps on lines "3,4,5" of which those on lines "5" never ran
    And shape-kept-blocked-lines-one-refusal: the second observation shows the scenario "S" failed with "AssertionError: protected output lacks repaired text", with steps on lines "3,4,5" of which those on lines "5" never ran
    When shape-kept-blocked-lines-one-refusal: the kept baseline refusals are computed
    Then shape-kept-blocked-lines-one-refusal: the only refusal contains "scenario \"S\" line 4 asserts unchanged behaviour but is red on the unchanged tree"
    And shape-kept-blocked-lines-one-refusal: the only refusal does not contain the word "lines"

  Scenario: Kept steps that are red or blocked in one observation only give no refusal
    Given shape-kept-blocked-lines-one-refusal: the property "P" names the kept steps on lines "4,5" of the scenario "S"
    And shape-kept-blocked-lines-one-refusal: the first observation shows the scenario "S" failed with "AssertionError: protected output lacks repaired text", with steps on lines "3,4,5" of which those on lines "5" never ran
    And shape-kept-blocked-lines-one-refusal: the second observation shows the scenario "S" passed
    When shape-kept-blocked-lines-one-refusal: the kept baseline refusals are computed
    Then shape-kept-blocked-lines-one-refusal: there are exactly 0 refusals

  Scenario: Different properties and different scenarios stay in separate refusals
    Given shape-kept-blocked-lines-one-refusal: the property "Alpha" names the kept steps on lines "3,4" of the scenario "S"
    And shape-kept-blocked-lines-one-refusal: the property "Beta" names the kept steps on lines "5,6" of the scenario "S"
    And shape-kept-blocked-lines-one-refusal: the property "Gamma" names the kept steps on lines "13,14" of the scenario "T"
    And shape-kept-blocked-lines-one-refusal: the first observation shows the scenario "S" failed with "AssertionError: protected output lacks repaired text", with steps on lines "2,3,4,5,6" of which those on lines "3,4,5,6" never ran
    And shape-kept-blocked-lines-one-refusal: the first observation shows the scenario "T" failed with "AssertionError: protected output lacks repaired text", with steps on lines "12,13,14" of which those on lines "13,14" never ran
    And shape-kept-blocked-lines-one-refusal: the second observation shows the scenario "S" failed with "AssertionError: protected output lacks repaired text", with steps on lines "2,3,4,5,6" of which those on lines "3,4,5,6" never ran
    And shape-kept-blocked-lines-one-refusal: the second observation shows the scenario "T" failed with "AssertionError: protected output lacks repaired text", with steps on lines "12,13,14" of which those on lines "13,14" never ran
    When shape-kept-blocked-lines-one-refusal: the kept baseline refusals are computed
    Then shape-kept-blocked-lines-one-refusal: there are exactly 3 refusals
    And shape-kept-blocked-lines-one-refusal: the refusal containing "- Alpha (kept)" contains "scenario \"S\" lines 3 and 4 is meant"
    And shape-kept-blocked-lines-one-refusal: the refusal containing "- Beta (kept)" contains "scenario \"S\" lines 5 and 6 is meant"
    And shape-kept-blocked-lines-one-refusal: the refusal containing "- Gamma (kept)" contains "scenario \"T\" lines 13 and 14 is meant"

  Scenario: A red kept step is not grouped with the blocked kept steps after it
    Given shape-kept-blocked-lines-one-refusal: the property "P" names the kept steps on lines "6,5,4,3" of the scenario "S"
    And shape-kept-blocked-lines-one-refusal: the first observation shows the scenario "S" failed with "AssertionError: protected output lacks repaired text", with steps on lines "3,4,5,6" of which those on lines "4,5,6" never ran
    And shape-kept-blocked-lines-one-refusal: the second observation shows the scenario "S" failed with "AssertionError: protected output lacks repaired text", with steps on lines "3,4,5" of which those on lines "4,5" never ran
    When shape-kept-blocked-lines-one-refusal: the kept baseline refusals are computed
    Then shape-kept-blocked-lines-one-refusal: there are exactly 2 refusals
    And shape-kept-blocked-lines-one-refusal: the refusal containing "line 3 asserts unchanged behaviour but is red" does not contain the word "lines"
    And shape-kept-blocked-lines-one-refusal: the refusal containing "lines 4 and 5 is meant to assert unchanged behaviour" does not contain the word "3"
    And shape-kept-blocked-lines-one-refusal: the refusal containing "lines 4 and 5 is meant to assert unchanged behaviour" does not contain the word "6"
