Feature: Every acceptance step line binds to exactly one definition before publication
  A scripted shaper submits the binding-proof package once through the real Shape entry point.
  The step self-check binds every concrete and expanded line with Cucumber's own matcher, without
  running a step body or hook, and refuses zero or several matches before readiness.

  Scenario: A doubly escaped RegExp leaves the code line unbound
    When k-shape-step-binding-proof: the scripted shaper submits the "escaped" binding-proof package
    Then k-shape-step-binding-proof: Shape refuses it before readiness with this feedback:
      """
      Step self-check failed; nothing was published. Fix the step files and resubmit.
      acceptance.feature line 5: undefined step: Then binding-proof: code 12
      """
    And k-shape-step-binding-proof: no step body or hook ran

  Scenario: The corrected RegExp binds every concrete and expanded line once
    When k-shape-step-binding-proof: the scripted shaper submits the "repaired" binding-proof package
    Then k-shape-step-binding-proof: the step self-check passes and readiness runs
    And k-shape-step-binding-proof: each of these step lines binds to exactly one definition:
      | binding-proof: input 1             |
      | binding-proof: I retain it         |
      | binding-proof: code 12             |
      | binding-proof: text "v2.4+[x]"     |
      | binding-proof: text "hello.world"  |
    And k-shape-step-binding-proof: the self-check warns of no unused definition
    And k-shape-step-binding-proof: no step body or hook ran

  Scenario: A duplicate definition makes the code line ambiguous
    When k-shape-step-binding-proof: the scripted shaper submits the "duplicate" binding-proof package
    Then k-shape-step-binding-proof: Shape refuses it before readiness with this feedback:
      """
      Step self-check failed; nothing was published. Fix the step files and resubmit.
      acceptance.feature line 5: ambiguous step: Then binding-proof: code 12 matches 2 definitions: ^binding-proof: code \d+$ in features/steps/binding-proof__steps.js; binding-proof: code {int} in features/steps/binding-proof__steps.js
      """
    And k-shape-step-binding-proof: no step body or hook ran

  Scenario: Both expanded Outline instances are reported unbound with their locations
    When k-shape-step-binding-proof: the scripted shaper submits the "text-deleted" binding-proof package
    Then k-shape-step-binding-proof: Shape refuses it before readiness with this feedback:
      """
      Step self-check failed; nothing was published. Fix the step files and resubmit.
      acceptance.feature line 9: undefined step: Then binding-proof: text "v2.4+[x]" (Examples row line 12)
      acceptance.feature line 9: undefined step: Then binding-proof: text "hello.world" (Examples row line 13)
      """
    And k-shape-step-binding-proof: no step body or hook ran

  Scenario: An unused definition only warns
    When k-shape-step-binding-proof: the scripted shaper submits the "unused" binding-proof package
    Then k-shape-step-binding-proof: the step self-check passes and readiness runs
    And k-shape-step-binding-proof: each of these step lines binds to exactly one definition:
      | binding-proof: input 1             |
      | binding-proof: I retain it         |
      | binding-proof: code 12             |
      | binding-proof: text "v2.4+[x]"     |
      | binding-proof: text "hello.world"  |
    And k-shape-step-binding-proof: the self-check warns only that "binding-proof: unused" in "features/steps/binding-proof__steps.js" is an unused definition
    And k-shape-step-binding-proof: no step body or hook ran

  Scenario: A step file that does not load is a failure to check
    When k-shape-step-binding-proof: the scripted shaper submits the "load-failure" binding-proof package
    Then k-shape-step-binding-proof: Shape refuses it before readiness with this feedback:
      """
      Step self-check failed; nothing was published. Fix the step files and resubmit.
      - load features/steps/binding-proof__steps.js: Error: cannot load definitions
      """
    And k-shape-step-binding-proof: the self-check reports no binding result
    And k-shape-step-binding-proof: no step body or hook ran

  Scenario: A separate token grammar binds exactly once
    When k-shape-step-binding-proof: the scripted shaper submits the "token" binding-proof package
    Then k-shape-step-binding-proof: the step self-check passes and readiness runs
    And k-shape-step-binding-proof: each of these step lines binds to exactly one definition:
      | binding-proof: input 1             |
      | binding-proof: I retain it         |
      | binding-proof: token a4b7c012      |
      | binding-proof: text "v2.4+[x]"     |
      | binding-proof: text "hello.world"  |
    And k-shape-step-binding-proof: the self-check warns of no unused definition
    And k-shape-step-binding-proof: no step body or hook ran
