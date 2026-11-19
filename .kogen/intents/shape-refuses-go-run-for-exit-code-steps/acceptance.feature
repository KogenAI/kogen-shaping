Feature: Shape step self-check refuses go run in exit-code steps
  This feature runs the step self-check on small step files and reads only its result.

  Scenario: A step asserting an exit code through go run is refused
    Given shape-refuses-go-run-for-exit-code-steps: the self-check ran on the fixture step files
    Then shape-refuses-go-run-for-exit-code-steps: the self-check of the "go run exit code" fixture refuses with the line "- exit_code features/steps/is-even__steps.js: step definition is-even: the program exits with code 2 asserts an exit code but obtains it through go run, which changes exit codes; build the binary (go build) and run it instead"

  Scenario: A step comparing stderr through go run is refused
    Given shape-refuses-go-run-for-exit-code-steps: the self-check ran on the fixture step files
    Then shape-refuses-go-run-for-exit-code-steps: the self-check of the "go run stderr" fixture refuses with the line "- exit_code features/steps/is-even__steps.js: step definition is-even: the program exits with code 2 asserts an exit code but obtains it through go run, which changes exit codes; build the binary (go build) and run it instead"

  Scenario: Each offending step of a file gets its own line
    Given shape-refuses-go-run-for-exit-code-steps: the self-check ran on the fixture step files
    Then shape-refuses-go-run-for-exit-code-steps: the self-check of the "go run two steps" fixture refuses with the lines "- exit_code features/steps/is-even__steps.js: step definition is-even: the program exits with code 2 asserts an exit code but obtains it through go run, which changes exit codes; build the binary (go build) and run it instead" and "- exit_code features/steps/is-even__steps.js: step definition is-even: the program exits with code 3 asserts an exit code but obtains it through go run, which changes exit codes; build the binary (go build) and run it instead"

  Scenario: A compiled binary step is accepted
    Given shape-refuses-go-run-for-exit-code-steps: the self-check ran on the fixture step files
    Then shape-refuses-go-run-for-exit-code-steps: the self-check of the "compiled binary" fixture accepts the step file

  Scenario: A step using go run only for output is accepted
    Given shape-refuses-go-run-for-exit-code-steps: the self-check ran on the fixture step files
    Then shape-refuses-go-run-for-exit-code-steps: the self-check of the "go run output only" fixture accepts the step file

  Scenario: A step file mentioning go run only in comments is accepted
    Given shape-refuses-go-run-for-exit-code-steps: the self-check ran on the fixture step files
    Then shape-refuses-go-run-for-exit-code-steps: the self-check of the "go run in a comment" fixture accepts the step file

  Scenario: A step mentioning go run only in a string nothing executes is accepted
    Given shape-refuses-go-run-for-exit-code-steps: the self-check ran on the fixture step files
    Then shape-refuses-go-run-for-exit-code-steps: the self-check of the "go run in a string" fixture accepts the step file
