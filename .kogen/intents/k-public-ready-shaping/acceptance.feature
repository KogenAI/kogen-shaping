Feature: Public repository documentation
  Scenario: Public readers receive usable instructions
    Then the public README contains:
      | text |
      | kogen-shaping |
      | https://github.com/KogenAI/kogen |
      | go build |
      | go test |
      | npm ci --prefix testdata/tools/cucumber |
      | npm ci --prefix testdata/tools/quint |
    And the public README states "contact@kogen.dev" exactly once
    And the public README omits private setup and internal paths
    And public policy files are absent

  Scenario: Owner receives an honest residual audit
    Then the public readiness audit contains:
      | text |
      | Current-tree findings |
      | History findings |
      | features/steps/testdata/k_models_file_shaping/recorded/models_landing_red.stderr |
      | shape/discovery_test.go |
    And the public readiness audit identifies historical commits
