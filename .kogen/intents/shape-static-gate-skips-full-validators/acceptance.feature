Feature: The static gate skips whole-repository validator scripts
  This feature asks the static gate's suite recognition about configured command lines and runs the
  gate with a recording command seam. A command that runs a script named validate_repo.py,
  validate_release.py or validate_all.py is a whole-repository check and is skipped like a test suite;
  other scripts and the commands recognised before keep their answers.

  Scenario: Whole-repository validator scripts are test suite checks
    Then shape-static-gate-skips-full-validators: each of these commands is a test suite check:
      """
      python3 reproduce/validate_repo.py
      python3 reproduce/validate_release.py --strict
      cd reproduce && python validate_all.py
      CI=1 bash ./validate_repo.py
      """

  Scenario: Other validator-like scripts are not test suite checks
    Then shape-static-gate-skips-full-validators: none of these commands is a test suite check:
      """
      python3 reproduce/validate_round.py
      python3 reproduce/validate_spec_snapshot.py
      python3 reproduce/check_shaped_files.py
      python3 reproduce/validate_repo.py.bak
      python3 reproduce/validate_repo_helpers.py
      """

  Scenario: The gate does not hand the validator to the command seam
    When shape-static-gate-skips-full-validators: the static gate runs the checks "validate-repo" and "privacy" with a recording command seam
    Then shape-static-gate-skips-full-validators: the only command given to the command seam is the "privacy" check command
    And shape-static-gate-skips-full-validators: the check "validate-repo" has the status "not_run"
    And shape-static-gate-skips-full-validators: the progress shows the line "kogen: static check validate-repo not run: it runs the test suite"

  Scenario: The gate still judges the shaped files with the other check
    When shape-static-gate-skips-full-validators: the static gate runs the checks "validate-repo" and "privacy" with a recording command seam
    Then shape-static-gate-skips-full-validators: the check "privacy" has the status "attributable"
    And shape-static-gate-skips-full-validators: the check "privacy" names the shaped file
    And shape-static-gate-skips-full-validators: the gate outcome is "refused"

  Scenario: Previously recognised commands keep their answers
    Then shape-static-gate-skips-full-validators: each of these commands is a test suite check:
      """
      go test ./...
      pytest -q
      bin/rails test
      make && (cargo test)
      """
    And shape-static-gate-skips-full-validators: none of these commands is a test suite check:
      """
      go build ./...
      bin/rails routes
      echo pytester
      """
