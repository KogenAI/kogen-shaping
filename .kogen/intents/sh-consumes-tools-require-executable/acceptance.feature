Feature: Consumer tool tests pass before and after the executable Tools.Require rule
  TestReadinessCheckAndKeptEvidence and TestToolsFromInstall in consumes/kogen-verification run
  with go test against the kogen-verification go.mod pins, and against a scratch copy of it with
  the new Tools.Require rule (node must be an executable file; a missing node is refused as
  "node is not available: node does not exist; reinstall Kogen"). They still fail against wrong
  producers.

  Scenario: Both tests pass against the pinned kogen-verification
    When sh-consumes-tools-require-executable: I run TestReadinessCheckAndKeptEvidence and TestToolsFromInstall against the "pinned" kogen-verification
    Then sh-consumes-tools-require-executable: both tests pass

  Scenario: Both tests pass against kogen-verification with the new Tools.Require rule and wording
    When sh-consumes-tools-require-executable: I run TestReadinessCheckAndKeptEvidence and TestToolsFromInstall against the "new Require" kogen-verification
    Then sh-consumes-tools-require-executable: both tests pass

  Scenario Outline: The tests still reject a wrong producer: <producer>
    When sh-consumes-tools-require-executable: I run TestReadinessCheckAndKeptEvidence and TestToolsFromInstall against the "<producer>" kogen-verification
    Then sh-consumes-tools-require-executable: "<test>" fails

    Examples:
      | producer                                             | test                              |
      | pinned with the node-path text altered               | TestToolsFromInstall              |
      | new Require with the node-path text altered          | TestToolsFromInstall              |
      | new Require without KOGEN_NODE in commands           | TestReadinessCheckAndKeptEvidence |
      | new Require resolving a missing node-path to /bin/sh | TestToolsFromInstall              |

  Scenario: The module pins are unchanged
    When sh-consumes-tools-require-executable: I read the checkout's go.mod and go.sum
    Then sh-consumes-tools-require-executable: go.mod requires exactly these modules, in this order:
      | module                                | version                            |
      | github.com/KogenAI/kogen-coordinator  | v0.0.0-20261105022408-e7bc7ce9c169 |
      | github.com/KogenAI/kogen-executor     | v0.0.0-20261105010505-8d465842f9df |
      | github.com/KogenAI/kogen-loop         | v0.0.0-20261105005409-31036b2c02e8 |
      | github.com/KogenAI/kogen-verification | v0.0.0-20261105003813-ac4f6a9a4877 |
    And sh-consumes-tools-require-executable: go.mod has no replace directive
    And sh-consumes-tools-require-executable: go.sum has the SHA-256 "ba20dbb2a5ce87b0077fc8b5843cbe58efd06ea32feb63fcf724c06a92c93229"
