Feature: TestKillGroupOnExit treats an exited but unreaped group member as gone
  consumes/kogen-loop's kill-group check counts a process of the command's group as running only
  while it runs: a group member that has exited but that its parent has not reaped yet (a zombie)
  is gone. A new test, skipped on platforms other than Linux, sets PR_SET_CHILD_SUBREAPER in a child
  process and runs the kill-group check there. On Linux every run goes through a parent process that
  is a child subreaper: one that never reaps the processes orphaned to it, as the Linux Build
  supervisor does, or one that reaps each at once, as init does. Other platforms have no child
  subreaper; there go test runs directly.

  Scenario: The kill-group checks pass while the killed group member stays an unreaped zombie
    When fix-k-shape-kill-group-zombie: I run "TestKillGroupOnExit" and the new test 1 time under a child-subreaper parent that never reaps orphaned processes
    Then fix-k-shape-kill-group-zombie: the go command exits 0
    And fix-k-shape-kill-group-zombie: "TestKillGroupOnExit" is reported as passed 1 time
    And fix-k-shape-kill-group-zombie: on Linux, "TestKillGroupOnExit" took at least 5 seconds, as the parent held its killed group member as a zombie
    And fix-k-shape-kill-group-zombie: the package has exactly one top-level test that the unchanged package does not have
    And fix-k-shape-kill-group-zombie: on Linux, the new test is reported as passed, each time after at least 5 seconds
    And fix-k-shape-kill-group-zombie: on other platforms, the new test is reported as skipped

  Scenario: The kill-group checks still fail when a group member keeps running
    Given fix-k-shape-kill-group-zombie: a scratch copy of the package that uses a kogen-loop whose KillGroupOnExit leaves the group running
    When fix-k-shape-kill-group-zombie: I run "TestKillGroupOnExit" and the new test 1 time under a child-subreaper parent that never reaps orphaned processes
    Then fix-k-shape-kill-group-zombie: the go command exits 1
    And fix-k-shape-kill-group-zombie: "TestKillGroupOnExit" is reported as failed with "of the command's group is still running after Run"
    And fix-k-shape-kill-group-zombie: the package has exactly one top-level test that the unchanged package does not have
    And fix-k-shape-kill-group-zombie: on Linux, the new test is reported as failed

  Scenario: The existing check passes without a holding subreaper, and the test process does not become one
    When fix-k-shape-kill-group-zombie: I run "TestKillGroupOnExit" and the new test 2 times under a child-subreaper parent that reaps every orphaned process at once
    Then fix-k-shape-kill-group-zombie: the go command exits 0
    And fix-k-shape-kill-group-zombie: "TestKillGroupOnExit" is reported as passed 2 times
    And fix-k-shape-kill-group-zombie: on Linux, every run of "TestKillGroupOnExit" took less than 5 seconds
    And fix-k-shape-kill-group-zombie: the package has exactly one top-level test that the unchanged package does not have
    And fix-k-shape-kill-group-zombie: on Linux, the new test is reported as passed

  Scenario: The package documents the Linux-only test and the kogen-loop follow-up
    When fix-k-shape-kill-group-zombie: I list the tests of the package
    Then fix-k-shape-kill-group-zombie: the inventory row of "sys.Options.KillGroupOnExit" in "consumes/kogen-loop/README.md" lists the new test
    And fix-k-shape-kill-group-zombie: the file "consumes/kogen-loop/README.md" does not contain "Nothing is skipped"
    And fix-k-shape-kill-group-zombie: a Go test file in "consumes/kogen-loop" names kogen-loop's own kill-group test as the follow-up
