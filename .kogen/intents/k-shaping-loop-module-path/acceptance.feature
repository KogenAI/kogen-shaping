Feature: kogen-shaping requires kogen-loop by its real module path
  kogen-shaping drops the bare kogen-loop requirement and its replace directive, requires
  github.com/KogenAI/kogen-loop at kogen-loop main, and moves kogen-coordinator, kogen-executor and
  kogen-verification to their main, so that no module in the build list imports kogen-loop/...

  Scenario: go.mod requires the Kogen components at their main without a replace
    When k-shaping-loop-module-path: I read the checkout's go.mod
    Then k-shaping-loop-module-path: go.mod requires exactly these modules, in this order:
      | module                                | version                            |
      | github.com/KogenAI/kogen-coordinator  | v0.0.0-20261105022408-e7bc7ce9c169 |
      | github.com/KogenAI/kogen-executor     | v0.0.0-20261105010505-8d465842f9df |
      | github.com/KogenAI/kogen-loop         | v0.0.0-20261105005409-31036b2c02e8 |
      | github.com/KogenAI/kogen-verification | v0.0.0-20261105003813-ac4f6a9a4877 |
    And k-shaping-loop-module-path: go.mod has no replace directive
    And k-shaping-loop-module-path: go.sum holds the module and go.mod hashes of each required version
    And k-shaping-loop-module-path: no go.work file is in effect for the checkout

  Scenario: No module in the build list is or imports the bare kogen-loop
    When k-shaping-loop-module-path: I list the build list and every package the module's packages and tests import
    Then k-shaping-loop-module-path: no module or package path is "kogen-loop" or starts with "kogen-loop/"
    And k-shaping-loop-module-path: the build list holds "github.com/KogenAI/kogen-loop v0.0.0-20261105005409-31036b2c02e8"
    And k-shaping-loop-module-path: the imported packages include:
      | github.com/KogenAI/kogen-loop/provider |
      | github.com/KogenAI/kogen-loop/sys      |
      | github.com/KogenAI/kogen-loop/tools    |
    And k-shaping-loop-module-path: no .js or .go.txt file directly under "features/steps" holds '"kogen-loop/'

  Scenario: The request's grep finds no bare kogen-loop import in Go files
    When k-shaping-loop-module-path: I run in the checkout:
      """
      grep -rn "\"kogen-loop/" --include=*.go .
      """
    Then k-shaping-loop-module-path: the command exits 1 and prints nothing

  Scenario: The Shaper policy changes only by the pinned kogen-loop's two tool descriptions
    Then k-shaping-loop-module-path: the file "shape/testdata/shaper_policy_golden.json" has SHA-256 "98c842c3fb874fc529e08dd3d0c1617575c974a1965c0924b3769061aa7e1e9c"
    And k-shaping-loop-module-path: the file "shape/testdata/prompts_golden.json" has SHA-256 "7b34f311c75d8c82f4afc0749c8fb36e6afd69d3f85473970e79fc27e023df41"
    And k-shaping-loop-module-path: the file "shape/testdata/defs_golden.json" has SHA-256 "c03bf24a45e5ffd4a1e5de8b38f6b8112a76528c35f74c21fba185ad41cbb8ec"

  Scenario: The module builds, vets and passes its tests on the real kogen-loop module path
    When k-shaping-loop-module-path: I read the checkout's go.mod
    Then k-shaping-loop-module-path: go.mod has no replace directive
    When k-shaping-loop-module-path: I run in the checkout:
      """
      go build ./... && go vet ./... && go test ./...
      """
    Then k-shaping-loop-module-path: the command exits 0
