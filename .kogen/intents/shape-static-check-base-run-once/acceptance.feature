Feature: The base-tree result of a static check is computed once and reused

  # The scenario shapes a request in a small project with three configured checks that fail with
  # and without the shaped files (exit 1, naming app.txt): "stale" and "twin" with the same command,
  # "other" with a different command. Every execution of a command without the shaped files appends
  # a line to a marker file. A scripted shaper submits a package, is refused on acceptance red,
  # submits a changed package (different step file bytes), is refused again, then acknowledges the
  # unchanged package. Each of the two submissions runs every check with the shaped files (a failing
  # check runs twice there: plain, then traced).

  Scenario: Each base-tree check runs once per check name and command across two refused submissions
    When shape-static-check-base-run-once: the project with the failing checks "stale", "twin" and "other" is shaped with two refused submissions
    Then shape-static-check-base-run-once: the Intent is published after the acknowledgment
    And shape-static-check-base-run-once: the command shared by the checks "stale" and "twin" ran without the shaped files 2 time(s)
    And shape-static-check-base-run-once: the command of the check "other" ran without the shaped files 1 time(s)
    And shape-static-check-base-run-once: the command of the check "other" ran with the shaped files 4 times, a plain and a traced run in each of the two rounds
    And shape-static-check-base-run-once: the progress lines of the failing static checks are, in order:
      """
      kogen: static check stale failed (exit 1) without the shaped files too: pre-existing failure in app.txt; the shaped files are not established by this check
      kogen: static check twin failed (exit 1) without the shaped files too: pre-existing failure in app.txt; the shaped files are not established by this check
      kogen: static check other failed (exit 1) without the shaped files too: pre-existing failure in app.txt; the shaped files are not established by this check
      kogen: static check stale failed (exit 1) without the shaped files too (base result reused): pre-existing failure in app.txt; the shaped files are not established by this check
      kogen: static check twin failed (exit 1) without the shaped files too (base result reused): pre-existing failure in app.txt; the shaped files are not established by this check
      kogen: static check other failed (exit 1) without the shaped files too (base result reused): pre-existing failure in app.txt; the shaped files are not established by this check
      """
