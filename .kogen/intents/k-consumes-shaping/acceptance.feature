Feature: kogen-shaping states exactly what it relies on from each Kogen dependency
  kogen-shaping adds one consumes/<producer>/ suite per Kogen dependency: kogen-coordinator,
  kogen-executor, kogen-loop and kogen-verification. Each suite is a Go package of test files
  written only against the producer's exported API; every use of the producer in kogen-shaping's
  own non-test code has a test that uses it and a row in the suite's README inventory, and the
  inventory lists nothing else. The suites are part of go test ./..., isolated from the caller's
  Git, HOME and module proxy, and pass at the dependency versions go.mod pins now.

  Scenario: Each Kogen dependency's consumes suite covers exactly the uses of it in kogen-shaping's own code
    When k-consumes-shaping: the consumes observer judges this checkout against these suites:
      | suite              | module                                 |
      | kogen-coordinator  | github.com/KogenAI/kogen-coordinator   |
      | kogen-executor     | github.com/KogenAI/kogen-executor      |
      | kogen-loop         | kogen-loop                             |
      | kogen-verification | github.com/KogenAI/kogen-verification  |
    Then k-consumes-shaping: the observer finds no problem
    And k-consumes-shaping: these paths are as committed at HEAD:
      | shape  |
      | cmd    |
      | go.mod |
      | go.sum |
    And k-consumes-shaping: the only main package outside "features" is "github.com/KogenAI/kogen-shaping/cmd/kogen-shape"

  Scenario: The consumes suites are part of the repository's normal test command and pass
    When k-consumes-shaping: the packages that "go test ./..." tests in this checkout are listed
    Then k-consumes-shaping: they include each of these packages:
      | github.com/KogenAI/kogen-shaping/consumes/kogen-coordinator  |
      | github.com/KogenAI/kogen-shaping/consumes/kogen-executor     |
      | github.com/KogenAI/kogen-shaping/consumes/kogen-loop         |
      | github.com/KogenAI/kogen-shaping/consumes/kogen-verification |
    When k-consumes-shaping: "go test -count=1 ./consumes/..." runs in this checkout
    Then k-consumes-shaping: go test passed and ran tests in each of these packages:
      | github.com/KogenAI/kogen-shaping/consumes/kogen-coordinator  |
      | github.com/KogenAI/kogen-shaping/consumes/kogen-executor     |
      | github.com/KogenAI/kogen-shaping/consumes/kogen-loop         |
      | github.com/KogenAI/kogen-shaping/consumes/kogen-verification |

  Scenario: The consumes suites pass isolated from the caller's Git, HOME and module proxy, under the race detector in shuffled order
    When k-consumes-shaping: "go test -race -count=1 -shuffle=on ./consumes/..." runs in this checkout with a bogus GIT_DIR, a fresh HOME that signs Git commits and no module proxy
    Then k-consumes-shaping: go test passed and ran tests in each of these packages:
      | github.com/KogenAI/kogen-shaping/consumes/kogen-coordinator  |
      | github.com/KogenAI/kogen-shaping/consumes/kogen-executor     |
      | github.com/KogenAI/kogen-shaping/consumes/kogen-loop         |
      | github.com/KogenAI/kogen-shaping/consumes/kogen-verification |
