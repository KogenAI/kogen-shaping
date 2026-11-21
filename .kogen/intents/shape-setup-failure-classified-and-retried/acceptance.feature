Feature: Shape setup failures are classified and retried once
  This feature observes scripted Shape runs whose configured setup is a scripted shell command, with no network. The clock is real time plus the scripted advances of the setup and the shaper request, so the 10 second pause is really waited.

  Background:
    Given shape-setup-failure-classified-and-retried: the supplied scripted setup observer is available
    When shape-setup-failure-classified-and-retried: the scripted shapings run

  Scenario: A setup that fails once and passes on the retry lets shaping continue
    Then shape-setup-failure-classified-and-retried: the "retry" observation equals
      """
      {"exit":0,"setup_runs":2,"setup_failure_events":1,"model_requests":2,"setup_ms_in_range":true,"elapsed_in_range":true}
      """

  Scenario: A setup that fails twice is refused as an environment failure with its command, exit code and last output lines
    Then shape-setup-failure-classified-and-retried: the "twice" observation equals
      """
      {"setup_runs":2,"setup_failure_events":2,"model_requests":0,"class":"environment","mentions_command":true,"mentions_exit_code":true,"first_output_line_in_text":"line 6","has_last_output_line":true,"has_line_before_last_20":false}
      """

  Scenario: A scripted setup of 90 seconds followed by a 100 second shaper request reports setup_ms 90000 and an elapsed time of 100 seconds
    Then shape-setup-failure-classified-and-retried: the "elapsed" observation equals
      """
      {"exit":0,"setup_runs":1,"setup_failure_events":0,"model_requests":2,"setup_ms_in_range":true,"elapsed_in_range":true}
      """
