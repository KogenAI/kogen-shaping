Feature: The controls time cap scales with the planned runs
  This feature checks the wrong-implementation check of a candidate whose runs take a faked amount
  of time on the injected clock. The time cap of a candidate is the larger of 300 seconds and 60
  seconds per planned run (the correct run plus the wrong implementations). A refusal and the
  wrong_implementation_check event state the cap that applied and the planned runs, a stop with a
  slow correct run says the project's own run is the slow part, and a finished run keeps its verdict.

  Background:
    Given shape-controls-time-cap-scales-with-runs: the time cap harness has run

  Scenario: Eight planned runs of fifty seconds are all judged under the 480 second cap
    Then shape-controls-time-cap-scales-with-runs: the observation of "eight-at-fifty" includes:
      """
      {"error":"","refusal":[],"runs":8,"event_result":"clean","event_runs":8,"time_cap_seconds":480,"planned_runs":8}
      """

  Scenario: A small candidate keeps the base cap of 300 seconds
    Then shape-controls-time-cap-scales-with-runs: the observation of "small" includes:
      """
      {"error":"","runs":2,"event_result":"refused","time_cap_seconds":300,"planned_runs":3,"refusal":["- time cap: the runs exceeded the total time cap of 300 seconds per candidate (3 planned runs; the cap is the larger of 300 seconds and 60 seconds per planned run) after 2 of 3 runs completed; the check stopped. The completed runs took 175 seconds on average. Make the implementations or the steps faster."]}
      """

  Scenario: A slow correct run is named as the project's own slow run
    Then shape-controls-time-cap-scales-with-runs: the observation of "slow" includes:
      """
      {"error":"","runs":5,"event_result":"refused","time_cap_seconds":480,"planned_runs":8,"refusal":["- time cap: the runs exceeded the total time cap of 480 seconds per candidate (8 planned runs; the cap is the larger of 300 seconds and 60 seconds per planned run) after 5 of 8 runs completed; the check stopped. The completed runs took 100 seconds on average. The project's own run is the slow part: the correct implementation's run alone took 300 seconds, more than half of the cap. Make the project's setup or acceptance command faster, or supply fewer wrong implementations."]}
      """

  Scenario: A wrong implementation that passed before the cap stopped the check is still reported
    Then shape-controls-time-cap-scales-with-runs: the observation of "verdicts" includes:
      """
      {"error":"","runs":5,"event_result":"refused","time_cap_seconds":360,"planned_runs":6,"refusal":["- time cap: the runs exceeded the total time cap of 360 seconds per candidate (6 planned runs; the cap is the larger of 300 seconds and 60 seconds per planned run) after 5 of 6 runs completed; the check stopped. The completed runs took 80 seconds on average. Make the implementations or the steps faster.","- add keeps working (kept) | request: add | problem: wrong implementation \"add loose\" passes scenario \"Add keeps working\""]}
      """
