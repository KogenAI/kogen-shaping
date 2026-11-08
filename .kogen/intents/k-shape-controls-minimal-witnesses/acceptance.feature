Feature: Control qualification runs the whole feature once and a minimal witness per wrong control
  The wrong-implementation check qualifies a candidate's controls.json: it runs the candidate's
  acceptance with the correct implementation and once with each wrong implementation. Today every
  one of those runs executes every scenario of the feature. After this change the correct
  implementation is run once against the whole feature, as the guard that every scenario can pass,
  and each wrong implementation is run only against the scenario (every example of an outline) that
  its controls.json entry names. The check still requires that each wrong implementation fails its
  target step with an AssertionError thrown by a step file, still rejects one that does not, and
  now records the target step and the assertion that rejected each control. The run cap of 8 and
  the time cap of 300 seconds do not change.

  Background:
    Given k-shape-controls-minimal-witnesses: the control qualification harness has run

  Scenario: The correct guard runs the whole feature once and each wrong control runs only the scenario it names
    Then k-shape-controls-minimal-witnesses: the observation of "clean" includes:
      """
      {"error":"","runs":3,"event_runs":3,"event_result":"clean","refusal":[],"executed":[["Add keeps working","Sub keeps working"],["Add keeps working"],["Sub keeps working"]]}
      """
    And k-shape-controls-minimal-witnesses: the observation of "outline" includes:
      """
      {"error":"","runs":3,"refusal":[],"executed":[["Add keeps working","Sub keeps working","Sub keeps working"],["Add keeps working"],["Sub keeps working","Sub keeps working"]]}
      """
    And k-shape-controls-minimal-witnesses: the observation of "target-scenario-only" includes:
      """
      {"error":"","runs":3,"event_result":"clean","refusal":[],"executed":[["Add keeps working","Sub keeps working"],["Add keeps working"],["Sub keeps working"]]}
      """

  Scenario: The qualification result records the target step and the assertion evidence of each control
    Then k-shape-controls-minimal-witnesses: the observation of "clean" includes:
      """
      {"controls":[{"name":"add removed","scenario":"Add keeps working","step":"Then is-even: add of 2 and 3 is 5","failed_step":"Then is-even: add of 2 and 3 is 5","assertion":"Expected values to be strictly equal:","location_file":"features/steps/is-even__steps.js","location_line":4},{"name":"sub removed","scenario":"Sub keeps working","step":"Then is-even: sub of 5 and 3 is 2","failed_step":"Then is-even: sub of 5 and 3 is 2","assertion":"Expected values to be strictly equal:","location_file":"features/steps/is-even__steps.js","location_line":4}]}
      """
    And k-shape-controls-minimal-witnesses: the observation of "earlier" includes:
      """
      {"controls":[{"name":"add removed","scenario":"Add keeps working","step":"And is-even: add of 2 and 3 gives an answer","failed_step":"Then is-even: calc takes 3 arguments","assertion":"Expected values to be strictly equal:","location_file":"features/steps/is-even__steps.js","location_line":11},{"name":"sub removed","scenario":"Sub keeps working","step":"Then is-even: sub of 5 and 3 is 2","failed_step":"Then is-even: sub of 5 and 3 is 2","assertion":"Expected values to be strictly equal:","location_file":"features/steps/is-even__steps.js","location_line":4}]}
      """
    And k-shape-controls-minimal-witnesses: the observation of "passes" includes:
      """
      {"controls":[{"name":"add removed","scenario":"Add keeps working","step":"Then is-even: add of 2 and 3 gives an answer","failed_step":"","assertion":"","location_file":"","location_line":0},{"name":"sub removed","scenario":"Sub keeps working","step":"Then is-even: sub of 5 and 3 is 2","failed_step":"Then is-even: sub of 5 and 3 is 2","assertion":"Expected values to be strictly equal:","location_file":"features/steps/is-even__steps.js","location_line":4}]}
      """

  Scenario: A wrong witness that does not fail its target step is still rejected
    Then k-shape-controls-minimal-witnesses: the observation of "passes" includes:
      """
      {"runs":3,"event_result":"refused","refusal":["- add keeps working (kept) | request: add | problem: wrong implementation \"add removed\" passes scenario \"Add keeps working\""]}
      """
    And k-shape-controls-minimal-witnesses: the observation of "earlier" includes:
      """
      {"runs":3,"event_result":"refused","refusal":["- add keeps working (kept) | request: add | problem: wrong implementation \"add removed\" fails scenario \"Add keeps working\" at \"Then is-even: calc takes 3 arguments\", not at \"And is-even: add of 2 and 3 gives an answer\""]}
      """

  Scenario: The run cap of 8 and the time cap of 300 seconds are unchanged
    Then k-shape-controls-minimal-witnesses: the observation of "earlier" includes:
      """
      {"headline_caps":true}
      """
    And k-shape-controls-minimal-witnesses: the observation of "run-cap" includes:
      """
      {"runs":0,"events":1,"event_result":"refused","refusal":["- run cap: this candidate needs 9 runs (the correct implementation and 8 wrong ones), above the cap of 8 runs per candidate; nothing was run. Supply at most 7 wrong implementations."]}
      """
