Feature: Refine refuses a draft for the same reasons Shape does
  kogen intent refine applies every acceptance check Shape applies before it publishes: the
  readiness gates (unwritten tests, the runner observer), the unpassable-step control run and the
  contract check. Refine refuses a refined draft that Shape would refuse, with Shape's exact
  feedback, and publishes a refined draft that passes them.

  Scenario: A refined draft whose acceptance runs tests the Build would invent is refused as Shape refuses it
    When k-refine-acceptance-gates: the "unwritten" journey runs
    Then k-refine-acceptance-gates: Refine judges the refined draft as Shape does

  Scenario: A refined draft whose step rejects the test runner's real output is refused as Shape refuses it
    When k-refine-acceptance-gates: the "observer" journey runs
    Then k-refine-acceptance-gates: Refine judges the refined draft as Shape does

  Scenario: A refined draft with a step no implementation can pass is refused as Shape refuses it
    When k-refine-acceptance-gates: the "unpassable" journey runs
    Then k-refine-acceptance-gates: Refine judges the refined draft as Shape does

  Scenario: A refined draft whose expectation contradicts its Intent is refused as Shape refuses it
    When k-refine-acceptance-gates: the "contract" journey runs
    Then k-refine-acceptance-gates: Refine judges the refined draft as Shape does

  Scenario: A refined draft whose step fails because the fresh copy is unprepared is refused as Shape refuses it
    When k-refine-acceptance-gates: the "environment" journey runs
    Then k-refine-acceptance-gates: Refine judges the refined draft as Shape does

  Scenario: A plain refinement that passes every check publishes after one refiner request
    When k-refine-acceptance-gates: the "plain" journey runs
    Then k-refine-acceptance-gates: Refine judges the refined draft as Shape does

  Scenario: A refined draft with ordinary missing-feature red is acknowledged and publishes
    When k-refine-acceptance-gates: the "red" journey runs
    Then k-refine-acceptance-gates: Refine judges the refined draft as Shape does

  Scenario: A refused refinement is repaired in the same run and publishes
    When k-refine-acceptance-gates: the "repair" journey runs
    Then k-refine-acceptance-gates: Refine judges the refined draft as Shape does
