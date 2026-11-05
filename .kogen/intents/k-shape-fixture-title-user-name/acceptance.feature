Feature: k-shape-fixture-title-user-name: Approve fixtures never title their Intent with the Git user name

  Scenario: The approve regression passes against the pinned coordinator
    Then k-shape-fixture-title-user-name: go.mod requires kogen-coordinator no older than "v0.0.0-20261103214825-03d7341feab7"
    When k-shape-fixture-title-user-name: go test runs "^TestApproveWithFullAndPrefixHash$" in "./shape"
    Then k-shape-fixture-title-user-name: each of these tests passes exactly once:
      | TestApproveWithFullAndPrefixHash    |
      | TestApproveWithFullAndPrefixHash/64 |
      | TestApproveWithFullAndPrefixHash/8  |
      | TestApproveWithFullAndPrefixHash/6  |

  Scenario: Fixtures that configure a Git user name title their Intent without it
    When k-shape-fixture-title-user-name: the module's fixtures are scanned for Git user names and Intent titles
    Then k-shape-fixture-title-user-name: the fixtures that configure a Git user name and build an Intent title are exactly these files:
      | consumes/kogen-verification/helpers_test.go                 |
      | features/steps/k-approve-binding-check-batched__harness.js |
      | shape/approve_test.go                                       |
    And k-shape-fixture-title-user-name: no such fixture titles its Intent with its Git user name

  Scenario: No kogen-shaping test asserts the coordinator title-lint wording or user-name rule
    When k-shape-fixture-title-user-name: the module's Go files and step files are searched for coordinator title-lint wording
    Then k-shape-fixture-title-user-name: none of them contains any of:
      | names the Git user              |
      | owner-authored title            |
      | narrates owner or user approval |
      | third person                    |
      | CheckTitle                      |
