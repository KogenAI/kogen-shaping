Feature: A controls.json patch with wrong hunk counts applies
  This feature covers the implementation overlay of a controls.json: a patch whose hunk header
  counts are one too small but whose context and changed lines match the tree applies and the
  implementation is judged as before, while a patch whose context line does not match the tree is
  still refused with the failed-hunk display.

  Background:
    Given shape-controls-patch-applies-with-recount: the controls patch harness has run

  Scenario: A patch whose hunk counts are one too small applies
    Then shape-controls-patch-applies-with-recount: the refusal lines of "miscounted" are:
      """
      []
      """

  Scenario: A patch with a context line that does not match the tree is still refused
    Then shape-controls-patch-applies-with-recount: the refusal lines of "changed-context" are:
      """
      ["- controls.json: the correct implementation: notes.txt: the patch does not apply to the candidate tree: error: patch failed: notes.txt:1: @@ -1,3 +1,3 @@: notes.txt from line 1 is: [1] apple [2] birch [3] cedar [4] dune [5] elm [6] fern [7] grove [8] heath"]
      """
