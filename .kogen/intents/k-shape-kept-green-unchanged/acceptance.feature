Feature: Kept scenarios agree with the unchanged tree
  Scenario: Refuse an incorrect pin then publish the correction
    When k-shape-kept-green-unchanged: Shape processes "correct"
    Then k-shape-kept-green-unchanged: the kept-output policy holds

  Scenario: Publish a green pin without extra provider requests
    When k-shape-kept-green-unchanged: Shape processes "green"
    Then k-shape-kept-green-unchanged: the kept-output policy holds

  Scenario: Refuse B but not new C
    When k-shape-kept-green-unchanged: Shape processes "mixed"
    Then k-shape-kept-green-unchanged: the kept-output policy holds

  Scenario: Leave missing fixtures to existing handling
    When k-shape-kept-green-unchanged: Shape processes "missing"
    Then k-shape-kept-green-unchanged: the kept-output policy holds

  Scenario: Preserve the plain provider sequence
    When k-shape-kept-green-unchanged: Shape processes "plain"
    Then k-shape-kept-green-unchanged: the kept-output policy holds

  Scenario: Do not refuse a transient red pin
    When k-shape-kept-green-unchanged: Shape processes "flaky"
    Then k-shape-kept-green-unchanged: the kept-output policy holds
