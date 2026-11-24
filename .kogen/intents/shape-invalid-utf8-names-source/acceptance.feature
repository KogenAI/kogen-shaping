Feature: Identify the source of invalid UTF-8 requests
  This feature preserves validation and the standalone adapter's JSON output contract.

  Scenario: Invalid UTF-8 from a request file
    Given shape-invalid-utf8-names-source: the standalone Shape command is built
    When shape-invalid-utf8-names-source: I supply bytes "ff" from "file"
    Then shape-invalid-utf8-names-source: stdout is exactly one JSON refusal with no Intent and exit code 2
    And shape-invalid-utf8-names-source: the error names the offending request file

  Scenario: Invalid UTF-8 from stdin
    Given shape-invalid-utf8-names-source: the standalone Shape command is built
    When shape-invalid-utf8-names-source: I supply bytes "ff" from "stdin"
    Then shape-invalid-utf8-names-source: stdout is exactly one JSON refusal with no Intent and exit code 2
    And shape-invalid-utf8-names-source: the error identifies stdin rather than a filesystem path
