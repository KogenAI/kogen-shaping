Feature: Refuse acceptance that does not cover the requested properties
  Before Shape publishes an Intent, Kogen identifies the properties the request and its answers ask
  for, including behaviours that must be kept, and refuses acceptance that leaves one unexercised,
  is red only because something is missing, counts builder-written tests, or claims a property only
  a person can judge. A plain input/output request keeps its route and its model requests.

  Scenario: Kept behaviours without a scenario are refused by name
    When k-shape-property-coverage: the "kept" case is shaped
    Then k-shape-property-coverage: candidate "tests only C" is refused naming "add keeps working" with "no scenario exercises it"
    And k-shape-property-coverage: candidate "tests only C" is refused naming "sub keeps working" with "no scenario exercises it"
    And k-shape-property-coverage: the first result for candidate "tests only C" does not contain "mul is memoized"
    And k-shape-property-coverage: candidate "tests only C" made 3 shaper requests and 2 contract check requests
    And k-shape-property-coverage: the contract review for candidate "tests only C" includes:
      """
      # Request

      Keep add(a, b) and sub(a, b) working exactly as they do now, and add mul(a, b), which returns the product of a and b.
      """
    And k-shape-property-coverage: candidate "tests A, B and C" publishes after a complete contract check
    And k-shape-property-coverage: the first result for candidate "tests A, B and C" contains "Scenario: Mul multiplies"
    And k-shape-property-coverage: the first result for candidate "tests A, B and C" does not contain "Scenario: Add keeps working ("
    And k-shape-property-coverage: the first result for candidate "tests A, B and C" does not contain "Scenario: Sub keeps working ("
    And k-shape-property-coverage: candidate "tests A, B and C" made 2 shaper requests and 1 contract check request

  Scenario: A bound checked only by counting builder-written tests is refused
    When k-shape-property-coverage: the "bound" case is shaped
    Then k-shape-property-coverage: candidate "counts builder-written tests" is refused naming "f is no slower than 100 ms" with "only checks that tests the Build writes exist or pass in some number"
    And k-shape-property-coverage: candidate "counts builder-written tests" made 3 shaper requests and 2 contract check requests
    And k-shape-property-coverage: candidate "measures f against the bound" publishes after a complete contract check
    And k-shape-property-coverage: the first result for candidate "measures f against the bound" contains "above 100 ms"
    And k-shape-property-coverage: candidate "measures f against the bound" made 2 shaper requests and 1 contract check request

  Scenario: A property step red only because its target file is missing is refused
    When k-shape-property-coverage: the "missing" case is shaped
    Then k-shape-property-coverage: candidate "target file absent" is refused naming "slug results are at most 20 characters" with "is red on the unchanged tree only because something it needs is missing"
    And k-shape-property-coverage: the first result for candidate "target file absent" contains "Cannot find module '../../slug'"
    And k-shape-property-coverage: candidate "target file absent" made 3 shaper requests and 2 contract check requests
    And k-shape-property-coverage: candidate "target file as a do-nothing form" publishes after a complete contract check
    And k-shape-property-coverage: the first result for candidate "target file as a do-nothing form" contains "slug returned"
    And k-shape-property-coverage: the first result for candidate "target file as a do-nothing form" does not contain "Cannot find module"
    And k-shape-property-coverage: candidate "target file as a do-nothing form" made 2 shaper requests and 1 contract check request

  Scenario: A property only a person can judge is recorded for human evaluation, not asserted
    When k-shape-property-coverage: the "human" case is shaped
    Then k-shape-property-coverage: candidate "records human evaluation" publishes after a complete contract check
    And k-shape-property-coverage: the contract review for candidate "records human evaluation" includes:
      """
      # Request

      Make the greeting feel welcoming.

      # Questions and answers

      Round 1:
      1. No automatic step can tell whether a greeting feels welcoming. How is that judged?
         a) Human judgement is fine: a person evaluates it
         b) Define a measurable rule instead
         answer: 1: a
      """
    And k-shape-property-coverage: candidate "records human evaluation" made 3 shaper requests and 1 contract check request
    And k-shape-property-coverage: candidate "asserts a welcoming score" is refused naming "the greeting feels welcoming" with "claims it holds"
    And k-shape-property-coverage: candidate "asserts a welcoming score" made 5 shaper requests and 2 contract check requests
    And k-shape-property-coverage: candidate "records no human evaluation" is refused naming "the greeting feels welcoming" with "no Decision records it as needing human evaluation"
    And k-shape-property-coverage: candidate "records no human evaluation" made 5 shaper requests and 2 contract check requests

  Scenario: A plain request publishes as before with the same model requests
    When k-shape-property-coverage: the "plain" case is shaped
    Then k-shape-property-coverage: candidate "one command, one output" publishes after a complete contract check
    And k-shape-property-coverage: candidate "one command, one output" made 2 shaper requests and 1 contract check request
    And k-shape-property-coverage: the first result for candidate "one command, one output" contains "Scenario: Hello prints hello"
    And k-shape-property-coverage: the first result for candidate "one command, one output" does not contain "Property check"
    And k-shape-property-coverage: the contract review for candidate "one command, one output" includes:
      """
      # Request

      Make node hello.js print hello.
      """
