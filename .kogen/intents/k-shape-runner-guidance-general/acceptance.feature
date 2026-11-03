Feature: General test-runner observer guidance without runner-specific examples

  Scenario: Shape's instructions carry only the general test-runner observer rule
    When k-shape-runner-guidance: the "guidance" journey runs
    Then k-shape-runner-guidance: the journey meets the general runner-observer contract

  Scenario: The runner-observer refusal names no runner and the gate refuses as before
    When k-shape-runner-guidance: the "refusal" journey runs
    Then k-shape-runner-guidance: the journey meets the general runner-observer contract
