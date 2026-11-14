Feature: Every strict tool Shape defines has a schema that OpenAI's strict mode accepts
  This feature covers the parameter schema of each tool Shape defines with strict true. In
  every object schema at any depth, each declared property is listed in required and the schema
  has additionalProperties false. The contract-review property report lists kept_steps as a
  required field, and a report that omits kept_steps is still accepted as an empty list.

  Scenario: Every strict tool's parameter schema is valid under the strict rules
    Given shape-strict-tool-schemas: the checkout as it is
    When shape-strict-tool-schemas: I run the observer tests of the strict tool schemas
    Then shape-strict-tool-schemas: the go command exits 0
    And shape-strict-tool-schemas: the observer test "TestShapeStrictToolSchemasObserver" is reported as passed

  Scenario: kept_steps is required in a property report and older reports without it are still accepted
    Given shape-strict-tool-schemas: the checkout as it is
    When shape-strict-tool-schemas: I run the observer tests of the strict tool schemas
    Then shape-strict-tool-schemas: the go command exits 0
    And shape-strict-tool-schemas: the observer test "TestShapeKeptStepsObserver" is reported as passed

  Scenario: The walking test rejects a nested property missing from required
    Given shape-strict-tool-schemas: a scratch overlay of the contract source file whose confirmation verdict item leaves "why" out of its required list
    When shape-strict-tool-schemas: I run the package tests whose names mention strict or schema
    Then shape-strict-tool-schemas: at least one of those tests is reported as failed and its output contains "why"

  Scenario: The walking test rejects an object schema without additionalProperties false
    Given shape-strict-tool-schemas: a scratch overlay of the contract source file whose confirmation verdict item has no additionalProperties false
    When shape-strict-tool-schemas: I run the package tests whose names mention strict or schema
    Then shape-strict-tool-schemas: at least one of those tests is reported as failed and its output contains "additionalProperties"
