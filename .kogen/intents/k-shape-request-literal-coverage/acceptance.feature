Feature: Shape checks that the candidate carries the required literal values of the request and answers

  # Each scenario runs one native Shape with a scripted shaper and a scripted literal-check model
  # (features/steps/k-shape-request-literal-coverage__harness.js). The request of the "omitted",
  # "substituted", "complete" and "unknown" cases is:
  #   Add a registration example to the README.
  #   The example must register exactly these two repositories: /demo_one and /demo-two.
  #   An earlier draft used /demo_old; it is superseded by /demo_one.
  #   Keep the wording friendly.
  # The "informal" case adds the line "For background only, an old prototype once used
  # /legacy-demo; it is not part of this change.". The "secret" case adds the line "Sign in with
  # password: hunter2-demo-secret.". The "answer" case asks one question and is answered "1: The example uses port 7443".

  Scenario: A candidate that only says it registers both repositories is refused naming both values, and the corrected package publishes
    When k-shape-request-literal-coverage: the "omitted" case is shaped
    Then k-shape-request-literal-coverage: the first submission is refused by the literal check with a line containing "literal: /demo_one", "source: request" and "status: missing"
    And k-shape-request-literal-coverage: the first submission is refused by the literal check with a line containing "literal: /demo-two", "source: request" and "status: missing"
    And k-shape-request-literal-coverage: the Intent is published from the corrected submission
    And k-shape-request-literal-coverage: the published intent.md contains "/demo_one"
    And k-shape-request-literal-coverage: the published intent.md contains "/demo-two"
    And k-shape-request-literal-coverage: exactly 2 literal check requests are made

  Scenario: A candidate that substitutes /demo-one for /demo_one is refused naming both values
    When k-shape-request-literal-coverage: the "substituted" case is shaped
    Then k-shape-request-literal-coverage: the first submission is refused by the literal check with a line containing "literal: /demo_one", "status: changed" and "package_value: /demo-one"
    And k-shape-request-literal-coverage: the first submission is refused by the literal check with no line containing "literal: /demo-two"
    And k-shape-request-literal-coverage: the Intent is published from the corrected submission

  Scenario: A complete candidate with a descriptive paraphrase publishes; superseded and unfound values are not demanded
    When k-shape-request-literal-coverage: the "complete" case is shaped
    Then k-shape-request-literal-coverage: exactly 1 literal check request is made
    And k-shape-request-literal-coverage: the literal check recorded "/demo_one" as "present"
    And k-shape-request-literal-coverage: the literal check recorded "/demo-two" as "present"
    And k-shape-request-literal-coverage: the literal check recorded "/demo_old" as "superseded"
    And k-shape-request-literal-coverage: the literal check recorded "/demo_three" as "unverified"
    And k-shape-request-literal-coverage: no feedback to the shaper contains "Literal check:"
    And k-shape-request-literal-coverage: the Intent is published from the first submission
    And k-shape-request-literal-coverage: the published intent.md is the submitted one and the Intent is not approved

  Scenario: The literal check reads the request, the answers and the whole candidate package
    When k-shape-request-literal-coverage: the "complete" case is shaped
    Then k-shape-request-literal-coverage: the literal check request carries the request, the answers and the candidate package
    And k-shape-request-literal-coverage: the literal check request includes:
      """
      A required literal is a value that the request or a settled answer explicitly requires the implementation or an example to use exactly, such as a path, command, name, identifier, number or example input; descriptive text that may be paraphrased is not a literal.
      """
    And k-shape-request-literal-coverage: the literal check request includes:
      """
      Give status superseded for a value that the request or a later answer explicitly replaces, and status secret for secret material such as a password, token, key or credential; for secret material leave literal, source_quote, package_value and why empty.
      """
    And k-shape-request-literal-coverage: the literal check request includes:
      """
      Copy each literal exactly as its source writes it, with source_quote copied exactly from that source and containing the literal; Kogen itself compares each literal with the package byte for byte.
      """
    And k-shape-request-literal-coverage: the literal check request includes:
      """
      Give status informal for a value that appears only in informal background, an aside or an example the source marks optional; Kogen never demands an informal value.
      """

  Scenario: A value that appears only in informal background is not demanded
    When k-shape-request-literal-coverage: the "informal" case is shaped
    Then k-shape-request-literal-coverage: the literal check recorded "/legacy-demo" as "informal"
    And k-shape-request-literal-coverage: no feedback to the shaper contains "Literal check:"
    And k-shape-request-literal-coverage: the Intent is published from the first submission

  Scenario: A value required by a settled answer is checked against the same candidate
    When k-shape-request-literal-coverage: the "answer" case is shaped
    Then k-shape-request-literal-coverage: the literal check request carries "answer 1: The example uses port 7443"
    And k-shape-request-literal-coverage: the first submission is refused by the literal check with a line containing "literal: 7443", "source: answer 1" and "status: missing"
    And k-shape-request-literal-coverage: the Intent is published from the corrected submission

  Scenario: An unavailable original request is reported as unknown, not compared
    When k-shape-request-literal-coverage: the "unknown" case is shaped
    Then k-shape-request-literal-coverage: the literal check recorded the original request as unknown
    And k-shape-request-literal-coverage: stderr contains "kogen: request literal check unknown: the original request is not available; required literals were not compared"
    And k-shape-request-literal-coverage: exactly 0 literal check requests are made
    And k-shape-request-literal-coverage: no feedback to the shaper contains "Literal check:"
    And k-shape-request-literal-coverage: the Intent is published from the first submission

  Scenario: Secret material in the request is neither demanded nor exposed
    When k-shape-request-literal-coverage: the "secret" case is shaped
    Then k-shape-request-literal-coverage: the first submission is refused by the literal check with a line containing "literal: /demo-two", "source: request" and "status: missing"
    And k-shape-request-literal-coverage: the literal check recorded 1 secret item
    And k-shape-request-literal-coverage: nothing Shape showed or recorded contains "hunter2-demo-secret"
    And k-shape-request-literal-coverage: the Intent is published from the corrected submission
