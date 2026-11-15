Feature: Refuse masked kept assertions before Build
  This feature reproduces studio-find-and-link-records (eb8361f8): scenario 3
  called new studio_records at line 45 before the kept assertion at line 51
  expected document.body instead of latest.body (lib/studio/document.ex:133).
  The draft reached Build and ended build_blocked. The reported 5 of 14 reds
  also included studio-concise-agent-intake and k-docs-concise-task-first;
  studio-issue-next-action expected "ran exactly once" while ExUnit printed
  "1 passed, 6 excluded". These local synthetic fixtures preserve those
  counterexamples without invoking Studio or pretending to run ExUnit.
  Studio issue: kogen/921054ec-feeb-46cf-8950-f8f3c955d2b1.
  Run counts include separate implementation controls when publication is eligible.

  Scenario: Reject a wrong document envelope after a new tool error
    When shape-kept-assertion-after-earlier-failure: Shape evaluates "error"
    Then shape-kept-assertion-after-earlier-failure: the unchanged-tree outcome is
      """
      {"published":false,"runs":2,"feedback":["Mixed document","line 4","never runs on the unchanged tree","studio_records"]}
      """

  Scenario: Reject a wrong summary after a new tool error
    When shape-kept-assertion-after-earlier-failure: Shape evaluates "summary"
    Then shape-kept-assertion-after-earlier-failure: the unchanged-tree outcome is
      """
      {"published":false,"runs":2,"feedback":["Mixed document","line 4","never runs on the unchanged tree","studio_records"]}
      """

  Scenario: Preserve refusal after an earlier new assertion is refuted
    When shape-kept-assertion-after-earlier-failure: Shape evaluates "assertion"
    Then shape-kept-assertion-after-earlier-failure: the unchanged-tree outcome is
      """
      {"published":false,"runs":2,"feedback":["Mixed document","line 4","never runs on the unchanged tree","studio_records"]}
      """

  Scenario: Allow missing new behaviour without a kept step
    When shape-kept-assertion-after-earlier-failure: Shape evaluates "new-only"
    Then shape-kept-assertion-after-earlier-failure: the unchanged-tree outcome is
      """
      {"published":true,"runs":3,"feedback":[]}
      """

  Scenario: Allow a passing kept step before missing new behaviour
    When shape-kept-assertion-after-earlier-failure: Shape evaluates "kept-first"
    Then shape-kept-assertion-after-earlier-failure: the unchanged-tree outcome is
      """
      {"published":true,"runs":4,"feedback":[]}
      """
