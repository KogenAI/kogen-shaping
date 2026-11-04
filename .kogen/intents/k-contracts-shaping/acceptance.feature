Feature: kogen-shaping pins what kogen uses of it in its own contract suite
  kogen-shaping's contract/ suite states, as tests, what its only consumer kogen uses: the
  exported shape.Shape, shape.Refine, shape.Approve and shape.Preflight with the inputs kogen
  sets, the exit codes kogen maps to its own, and the response-budget behaviour kogen's
  response_budget_test.go requires (TestShapeBudgetRolesShape). The cases are copied from
  kogen-shaping's own tests and made self-contained with synthetic fixtures and an in-process
  scripted provider. The suite runs in kogen-shaping's ordinary go test, so a change to that
  behaviour fails in kogen-shaping first. Production code stays as committed.

  Scenario: The contract suite passes in kogen-shaping's own gate under the race detector
    When k-contracts-shaping: "go test -race -count=1 ./contract/..." runs in this checkout
    Then k-contracts-shaping: go test passed and ran tests in package "github.com/KogenAI/kogen-shaping/contract"
    And k-contracts-shaping: package "github.com/KogenAI/kogen-shaping/contract" has test files and no other Go files
    And k-contracts-shaping: package "github.com/KogenAI/kogen-shaping/contract" imports none of:
      | net/http/httptest |
      | net               |
    And k-contracts-shaping: "contract/README.md" names each of:
      | shape.Shape               |
      | shape.Refine              |
      | shape.Approve             |
      | shape.Preflight           |
      | TestShapeBudgetRolesShape |
    And k-contracts-shaping: these paths are as committed at HEAD:
      | shape  |
      | cmd    |
      | go.mod |
      | go.sum |

  Scenario: The contract suite is fast enough to join kogen-shaping's ordinary check
    When k-contracts-shaping: "go test -count=1 ./contract/..." runs in this checkout
    Then k-contracts-shaping: go test reported package "github.com/KogenAI/kogen-shaping/contract" passing within 60 seconds

  Scenario Outline: The contract suite fails when kogen-shaping changes what kogen relies on: <change>
    When k-contracts-shaping: the change "<change>" is planted in a scratch copy of this checkout
    Then k-contracts-shaping: every package of the scratch copy builds and vets
    And k-contracts-shaping: "go test -count=1 ./contract/..." fails in the scratch copy because a test failed

    Examples:
      | change                                   |
      | shape records its response budget twice  |
      | refine records its response budget twice |
      | shape sends under a ten-minute budget    |
      | shape returns exit 1 for every refusal   |
      | refine refuses with exit 1               |
