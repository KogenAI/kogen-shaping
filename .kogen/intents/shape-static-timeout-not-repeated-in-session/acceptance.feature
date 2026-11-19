Feature: A static check that timed out is not run again in the same Shape session

  # Every scenario shapes a request through the real Shape entry point with a scripted shaper in a
  # small project whose configured checks are fakes. Round N is the Nth submission of a session. A
  # session submits different packages (one per round) and then acknowledges the unchanged last
  # package (whose static result is reused).

  Scenario: A check that timed out in round 1 is not started in round 2 and is recorded as unavailable with wall_ms 0
    When shape-static-timeout-not-repeated-in-session: the project for not-repeated is shaped
    Then shape-static-timeout-not-repeated-in-session: the Intent is published in session 1
    And shape-static-timeout-not-repeated-in-session: after session 1 the check "slow" was started 1 time in total
    And shape-static-timeout-not-repeated-in-session: in session 1 the static checks event 1 lists the check "slow" as "unavailable" with wall_ms greater than 0
    And shape-static-timeout-not-repeated-in-session: in session 1 the static checks event 2 lists the check "slow" as "unavailable" with wall_ms 0
    And shape-static-timeout-not-repeated-in-session: in session 1 the static checks event 2 gives the check "slow" the reason "timed out after 120 s in round 1; not run again in this session"
    And shape-static-timeout-not-repeated-in-session: the progress shows the line "kogen: static check slow unavailable: timed out after 120 s in round 1; not run again in this session"
    And shape-static-timeout-not-repeated-in-session: the static-checks record lists the check "slow" as "unavailable" with wall_ms 0

  Scenario: A check that was not found in round 1 is run again in round 2 and reports its new result
    When shape-static-timeout-not-repeated-in-session: the project for only-timeouts is shaped
    Then shape-static-timeout-not-repeated-in-session: the Intent is published in session 1
    And shape-static-timeout-not-repeated-in-session: after session 1 the check "tool" was started 2 times in total
    And shape-static-timeout-not-repeated-in-session: in session 1 the static checks event 1 lists the check "tool" as "unavailable" with wall_ms greater than 0
    And shape-static-timeout-not-repeated-in-session: in session 1 the static checks event 2 lists the check "tool" as "passed" with wall_ms greater than 0

  Scenario: A failed check is run in both rounds while a timed out check beside it is skipped in round 2
    When shape-static-timeout-not-repeated-in-session: the project for keeps-failed is shaped
    Then shape-static-timeout-not-repeated-in-session: the Intent is published in session 1
    And shape-static-timeout-not-repeated-in-session: in session 1 the static checks event 1 lists the check "bad" as "preexisting" with wall_ms greater than 0
    And shape-static-timeout-not-repeated-in-session: in session 1 the static checks event 2 lists the check "bad" as "passed" with wall_ms greater than 0
    And shape-static-timeout-not-repeated-in-session: in session 1 the static checks event 2 lists the check "slow" as "unavailable" with wall_ms 0
    And shape-static-timeout-not-repeated-in-session: after session 1 the check "slow" was started 1 time in total

  Scenario: A timed out check is skipped while mix.lock is unchanged and runs again once mix.lock changed
    When shape-static-timeout-not-repeated-in-session: the project for lock-change is shaped
    Then shape-static-timeout-not-repeated-in-session: the Intent is published in session 1
    And shape-static-timeout-not-repeated-in-session: in session 1 the static checks event 2 lists the check "slow" as "unavailable" with wall_ms 0
    And shape-static-timeout-not-repeated-in-session: in session 1 the static checks event 3 lists the check "slow" as "unavailable" with wall_ms greater than 0
    And shape-static-timeout-not-repeated-in-session: in session 1 the static checks event 3 gives the check "slow" a reason without "not run again"
    And shape-static-timeout-not-repeated-in-session: after session 1 the check "slow" was started 2 times in total

  Scenario: A new session on the same tree starts with no memory
    When shape-static-timeout-not-repeated-in-session: the project for new-session is shaped
    Then shape-static-timeout-not-repeated-in-session: the Intent is published in session 2
    And shape-static-timeout-not-repeated-in-session: after session 1 the check "slow" was started 1 time in total
    And shape-static-timeout-not-repeated-in-session: after session 2 the check "slow" was started 2 times in total
    And shape-static-timeout-not-repeated-in-session: in session 2 the static checks event 1 lists the check "slow" as "unavailable" with wall_ms greater than 0
    And shape-static-timeout-not-repeated-in-session: in session 2 the static checks event 1 gives the check "slow" a reason without "not run again"
    And shape-static-timeout-not-repeated-in-session: in session 2 the static checks event 2 lists the check "slow" as "unavailable" with wall_ms 0
