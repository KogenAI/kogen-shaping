Feature: An inapplicable controls patch shows the real text

  This feature checks the refusal line that Shape reports when a patch in controls.json does
  not apply to the candidate tree. It runs the real controls check on small fixture trees.

  Scenario: A correct-implementation patch whose context differs shows the file lines
    Given shape-reports-inapplicable-control-lines: the candidate tree has the file "notes.txt" with these lines:
      """
      apple
      birch
      cedar
      dune
      elm
      fern
      grove
      heath
      iris
      juniper
      kelp
      larch
      maple
      nettle
      oak
      pine
      quince
      reed
      spruce
      thistle
      """
    When shape-reports-inapplicable-control-lines: the correct implementation patches "notes.txt" with:
      """
      --- a/notes.txt
      +++ b/notes.txt
      @@ -10,5 +10,5 @@
       juniper
       kelp
       DIFFERENT
      -maple
      +MAPLE
       nettle
      """
    Then shape-reports-inapplicable-control-lines: the refusal line starts with:
      """
      - controls.json: the correct implementation: notes.txt: the patch does not apply to the candidate tree: error: patch failed: notes.txt:10
      """
    And shape-reports-inapplicable-control-lines: the refusal line then names the hunk header "@@ -10,5 +10,5 @@"
    And shape-reports-inapplicable-control-lines: the refusal line then shows numbered lines 10 to 17 of the file in order and no others

  Scenario: A wrong-implementation patch keeps its wording and adds the same excerpt
    Given shape-reports-inapplicable-control-lines: the candidate tree has the file "notes.txt" with these lines:
      """
      apple
      birch
      cedar
      dune
      elm
      fern
      grove
      heath
      iris
      juniper
      kelp
      larch
      maple
      nettle
      oak
      pine
      quince
      reed
      spruce
      thistle
      """
    When shape-reports-inapplicable-control-lines: a wrong implementation patches "notes.txt" with:
      """
      --- a/notes.txt
      +++ b/notes.txt
      @@ -10,5 +10,5 @@
       juniper
       kelp
       DIFFERENT
      -maple
      +MAPLE
       nettle
      """
    Then shape-reports-inapplicable-control-lines: the refusal line starts with:
      """
      - controls.json: wrong implementation "add removed": notes.txt: the patch does not apply to the candidate tree with the correct implementation applied: error: patch failed: notes.txt:10
      """
    And shape-reports-inapplicable-control-lines: the refusal line then names the hunk header "@@ -10,5 +10,5 @@"
    And shape-reports-inapplicable-control-lines: the refusal line then shows numbered lines 10 to 17 of the file in order and no others

  Scenario: A patch for a file that is not in the candidate tree says so
    Given shape-reports-inapplicable-control-lines: the candidate tree has no file "missing.txt"
    When shape-reports-inapplicable-control-lines: the correct implementation patches "missing.txt" with:
      """
      --- a/missing.txt
      +++ b/missing.txt
      @@ -2,3 +2,3 @@
       birch
      -cedar
      +CEDAR
       dune
      """
    Then shape-reports-inapplicable-control-lines: the refusal line starts with:
      """
      - controls.json: the correct implementation: missing.txt: the patch does not apply to the candidate tree: error: missing.txt: No such file or directory
      """
    And shape-reports-inapplicable-control-lines: the refusal line then names the hunk header "@@ -2,3 +2,3 @@"
    And shape-reports-inapplicable-control-lines: the refusal line then says the file does not exist in the candidate tree and shows no numbered lines

  Scenario: A hunk near the end of a short file shows only the lines that exist
    Given shape-reports-inapplicable-control-lines: the candidate tree has the file "short.txt" with these lines:
      """
      apple
      birch
      cedar
      dune
      elm
      """
    When shape-reports-inapplicable-control-lines: the correct implementation patches "short.txt" with:
      """
      --- a/short.txt
      +++ b/short.txt
      @@ -3,3 +3,3 @@
       cedar
       DIFFERENT
      -elm
      +ELM
      """
    Then shape-reports-inapplicable-control-lines: the refusal line starts with:
      """
      - controls.json: the correct implementation: short.txt: the patch does not apply to the candidate tree: error: patch failed: short.txt:3
      """
    And shape-reports-inapplicable-control-lines: the refusal line then names the hunk header "@@ -3,3 +3,3 @@"
    And shape-reports-inapplicable-control-lines: the refusal line then shows numbered lines 3 to 5 of the file in order and no others

  Scenario: A patch that applies cleanly produces no refusal line
    Given shape-reports-inapplicable-control-lines: the candidate tree has the file "notes.txt" with these lines:
      """
      apple
      birch
      cedar
      dune
      elm
      """
    When shape-reports-inapplicable-control-lines: the correct implementation patches "notes.txt" with:
      """
      --- a/notes.txt
      +++ b/notes.txt
      @@ -2,3 +2,3 @@
       birch
      -cedar
      +CEDAR
       dune
      """
    Then shape-reports-inapplicable-control-lines: no refusal line is reported
