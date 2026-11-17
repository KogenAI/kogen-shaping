Feature: The shaper prompt requires a scaling scenario for collection operations
  This feature covers the instructions Shape gives the shaper. When a request describes an
  operation over a collection of records of unknown size, the shaper must write one acceptance
  scenario that runs the operation with two different record counts and bounds the work it does
  against its data store or external services, or must skip it with a stated reason in the Tests
  section. The rendered shaper prompt must carry that rule and the recorded prompt bytes must match.

  Scenario: The shaper prompt carries the collection scaling rule
    Given shape-collection-features-get-scaling-scenario: the rendered shaper prompt is loaded
    Then shape-collection-features-get-scaling-scenario: the rendered shaper prompt has this line:
      """
      - Work over a collection. When the request describes an operation over a collection of records of unknown size (for example an export, an import, a bulk update, a listing, a report or a sync), the acceptance package includes one scenario that runs the operation twice with different record counts (for example 10 and 100) and asserts that the work the operation does against its data store or external services (for example the number of queries counted through the framework's own instrumentation, or the calls counted by a counting stub) stays within the same stated upper bound for both counts, so that the work does not grow with the record count. The scenario uses the project's own test tooling, and the controls.json wrong implementation for it does that work once per record. Skip this scenario only with a `- none: <reason>` line in `## Tests` that says why the operation cannot grow with the record count. A request that does not process a collection gets no such scenario.
      """
    And shape-collection-features-get-scaling-scenario: the prompts golden records the rendered shaper prompt
