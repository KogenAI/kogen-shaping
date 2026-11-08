Feature: Refuse acceptance steps that run whole test suites
  A scripted shaper submits one package through the real Shape entry point. The step self-check reads
  the source of the step definitions the feature uses, before readiness and before any acceptance run.
  It refuses a step that runs a whole test suite and names the step and the step limit that approval
  uses; steps that run one named test or one package are accepted. No step body ever runs.

  Background:
    Given shape-refuses-steps-over-step-limit: the shape acceptance harness is compiled

  Scenario Outline: A step that runs a whole test suite is refused naming the step and the limit
    When shape-refuses-steps-over-step-limit: the scripted shaper submits the "<case>" step-limit package
    Then shape-refuses-steps-over-step-limit: Shape refuses it before readiness with this feedback:
      """
      Step self-check failed; nothing was published. Fix the step files and resubmit.
      - step_limit limit-proof: <step>: runs a whole test suite with <runner> and no narrowing; narrow it to one named test or package, or split the step, so it finishes below the 120 s step limit
      """
    And shape-refuses-steps-over-step-limit: no step body ran

    Examples:
      | case       | step                                               | runner   |
      | go-shell   | I run the Go suite                                 | go test  |
      | go-argv    | I run the Go suite with an argument list           | go test  |
      | concurrent | I run two uncached full Go suites concurrently     | go test  |
      | mix        | I run the Elixir suite                             | mix test |
      | npm        | I run the JavaScript suite                         | npm test |
      | mixed      | I run the whole Go suite                           | go test  |

  Scenario Outline: A step that runs one named test or one package is accepted
    When shape-refuses-steps-over-step-limit: the scripted shaper submits the "<case>" step-limit package
    Then shape-refuses-steps-over-step-limit: the step self-check passes and readiness runs
    And shape-refuses-steps-over-step-limit: the self-check reports no step over the step limit
    And shape-refuses-steps-over-step-limit: no step body ran

    Examples:
      | case              |
      | narrow-go-test    |
      | narrow-go-package |
      | narrow-mix        |
      | narrow-npm        |

  Scenario: The refusal uses the step limit that approval uses
    When shape-refuses-steps-over-step-limit: the scripted shaper submits the "concurrent" step-limit package
    Then shape-refuses-steps-over-step-limit: the feedback gives the 120 s limit that verify.StepTimeout holds
    And shape-refuses-steps-over-step-limit: no Go file of the shape package outside its tests holds a 120-second limit of its own
