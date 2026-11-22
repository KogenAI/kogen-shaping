Feature: Observer control mismatches show the control's stdout

  Scenario: Stdout of a mismatching control follows the exit line
    Given shape-observer-mismatch-shows-stdout: the good control is "echo 'usage: measure DIR PATH'; exit 2"
    And shape-observer-mismatch-shows-stdout: the bad control is "exit 1"
    When shape-observer-mismatch-shows-stdout: the observer controls are re-run
    Then shape-observer-mismatch-shows-stdout: the good item is not ok with detail:
      """
      exit 2, recorded exit 0
      stdout: usage: measure DIR PATH
      """

  Scenario: Exit, stderr and stdout lines come in that order
    Given shape-observer-mismatch-shows-stdout: the good control is "echo 'from stdout'; echo 'from stderr' >&2; exit 3"
    And shape-observer-mismatch-shows-stdout: the bad control is "exit 1"
    When shape-observer-mismatch-shows-stdout: the observer controls are re-run
    Then shape-observer-mismatch-shows-stdout: the good item is not ok with detail:
      """
      exit 3, recorded exit 0
      stderr: from stderr
      stdout: from stdout
      """

  Scenario: Stdout is redacted before its last 4096 bytes are taken
    Given shape-observer-mismatch-shows-stdout: the good control is "printf HEADMARK; head -c 879 /dev/zero | tr '\\0' p; printf sos-secret-value-123456; head -c 4083 /dev/zero | tr '\\0' x; printf TAILEND; exit 2"
    And shape-observer-mismatch-shows-stdout: the bad control is "exit 1"
    When shape-observer-mismatch-shows-stdout: the observer controls are re-run
    Then shape-observer-mismatch-shows-stdout: the good item detail starts with "exit 2, recorded exit 0\nstdout: "
    And shape-observer-mismatch-shows-stdout: the good item stdout line is valid text of at most 4096 bytes ending with "TAILEND"
    And shape-observer-mismatch-shows-stdout: the good item detail holds none of "HEADMARK" and "value-123456"

  Scenario: Matching exits and empty stdout add nothing
    Given shape-observer-mismatch-shows-stdout: the good control is "echo 'printed but matching'"
    And shape-observer-mismatch-shows-stdout: the bad control is "echo printed; exit 1"
    When shape-observer-mismatch-shows-stdout: the observer controls are re-run
    Then shape-observer-mismatch-shows-stdout: the good item is ok with empty detail
    And shape-observer-mismatch-shows-stdout: the bad item is ok with empty detail

  Scenario: A mismatch with empty stdout has only the exit line
    Given shape-observer-mismatch-shows-stdout: the good control is "exit 7"
    And shape-observer-mismatch-shows-stdout: the bad control is "exit 1"
    When shape-observer-mismatch-shows-stdout: the observer controls are re-run
    Then shape-observer-mismatch-shows-stdout: the good item is not ok with detail:
      """
      exit 7, recorded exit 0
      """

  Scenario: Each control has its own stdout
    Given shape-observer-mismatch-shows-stdout: the good control is "echo 'first control'; exit 2"
    And shape-observer-mismatch-shows-stdout: the bad control is "echo 'second control'; exit 4"
    When shape-observer-mismatch-shows-stdout: the observer controls are re-run
    Then shape-observer-mismatch-shows-stdout: the good item is not ok with detail:
      """
      exit 2, recorded exit 0
      stdout: first control
      """
    And shape-observer-mismatch-shows-stdout: the bad item is not ok with detail:
      """
      exit 4, recorded exit 1
      stdout: second control
      """

  Scenario: On a timeout the stdout line comes before the timed out sentence
    Given shape-observer-mismatch-shows-stdout: the probe deadline is 3000 milliseconds
    And shape-observer-mismatch-shows-stdout: the good control is "echo 'before timeout'; sleep 60"
    And shape-observer-mismatch-shows-stdout: the bad control is "exit 1"
    When shape-observer-mismatch-shows-stdout: the observer controls are re-run
    Then shape-observer-mismatch-shows-stdout: the good item detail holds "stdout: before timeout" before ", timed out"
