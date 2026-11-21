Feature: A failing reference gets a repair round
  This feature observes scripted Shape runs with a fake client, a fake clock and scripted checks, with no network.

  Background:
    Given shape-failing-reference-gets-repair-round: the supplied scripted Shape observer is available
    When shape-failing-reference-gets-repair-round: the scripted shapings run

  Scenario: A reference failure at 1300 seconds names the command and exit code and gets a second request
    Then shape-failing-reference-gets-repair-round: the "named" observation equals
      """
      {"requests":2,"names_command":true,"names_exit_code":true,"names_assertion_lines":true,"says_reference_or_test_at_fault":true}
      """

  Scenario: A second request that fixes the reference passes
    Then shape-failing-reference-gets-repair-round: the "repaired" observation equals
      """
      {"exit":0,"failed":false,"shaper_requests_at_least_two":true}
      """

  Scenario: A second request that does not fix the reference is refused with the extra line after two requests
    Then shape-failing-reference-gets-repair-round: the "unrepaired" observation equals
      """
      {"failed":true,"requests":2,"ends_with_extra_line":true,"extra_line_once":true}
      """

  Scenario: A refusal that is not about the reference gets no extra round
    Then shape-failing-reference-gets-repair-round: the "other" observation equals
      """
      {"failed":true,"requests":3,"mentions_extra_line":false}
      """
