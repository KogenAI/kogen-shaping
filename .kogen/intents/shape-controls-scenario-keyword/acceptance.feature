Feature: A controls.json scenario field may repeat the Gherkin keyword
  This feature covers how Shape parses the scenario field of a controls.json wrong implementation
  against a fixture feature with the scenarios Add keeps working, the outline Mul with two examples,
  a scenario named Scenario: odd and a scenario named odd. Each observation holds the problem lines,
  the stored scenario name and the witness indices of the parsed control.

  Background:
    Given shape-controls-scenario-keyword: the scenario keyword harness has run

  Scenario: A scenario field with the Scenario keyword resolves to the plain name
    Then shape-controls-scenario-keyword: the observation of "plain scenario" is:
      """
      {"problems": [], "scenario": "Add keeps working", "witness": [0]}
      """
    Then shape-controls-scenario-keyword: the observation of "keyword scenario" is:
      """
      {"problems": [], "scenario": "Add keeps working", "witness": [0]}
      """

  Scenario: A scenario field with the Scenario Outline keyword designates every example of the outline
    Then shape-controls-scenario-keyword: the observation of "plain outline" is:
      """
      {"problems": [], "scenario": "Mul <a> times <b>", "witness": [1, 2]}
      """
    Then shape-controls-scenario-keyword: the observation of "keyword outline" is:
      """
      {"problems": [], "scenario": "Mul <a> times <b>", "witness": [1, 2]}
      """
    Then shape-controls-scenario-keyword: the observation of "keyword template" is:
      """
      {"problems": [], "scenario": "Mul <a> times <b>", "witness": [1, 2]}
      """

  Scenario: A name that matches nothing is refused with and without a keyword
    Then shape-controls-scenario-keyword: the observation of "unknown plain" is:
      """
      {"problems": ["- controls.json: wrong implementation \"n1\": scenario \"Missing\" is not a scenario of acceptance.feature"], "scenario": "", "witness": []}
      """
    Then shape-controls-scenario-keyword: the observation of "unknown keyword" is:
      """
      {"problems": ["- controls.json: wrong implementation \"n1\": scenario \"Scenario: Missing\" is not a scenario of acceptance.feature"], "scenario": "", "witness": []}
      """

  Scenario: A scenario named with the keyword text resolves to itself and the plain name to the other
    Then shape-controls-scenario-keyword: the observation of "odd with keyword text" is:
      """
      {"problems": [], "scenario": "Scenario: odd", "witness": [3]}
      """
    Then shape-controls-scenario-keyword: the observation of "odd plain" is:
      """
      {"problems": [], "scenario": "odd", "witness": [4]}
      """

  Scenario: The keyword is removed once
    Then shape-controls-scenario-keyword: the observation of "keyword twice" is:
      """
      {"problems": ["- controls.json: wrong implementation \"n1\": scenario \"Scenario: Scenario: Add keeps working\" is not a scenario of acceptance.feature"], "scenario": "", "witness": []}
      """
    Then shape-controls-scenario-keyword: the observation of "keyword twice before keyword-named" is:
      """
      {"problems": [], "scenario": "Scenario: odd", "witness": [3]}
      """
