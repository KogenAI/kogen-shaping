Feature: Self-check stderr refusal diagnostics
  Scenario: Failing observer stderr survives retries without leaking secrets
    When k-shape-selfcheck-stderr: observer refusal is exhausted
    Then k-shape-selfcheck-stderr: the redacted stderr tail is visible
