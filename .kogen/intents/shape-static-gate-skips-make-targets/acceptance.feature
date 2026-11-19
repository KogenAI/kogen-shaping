Feature: The static gate skips make targets that verify the whole project
  This feature asks the static gate's suite recognition about configured command lines and runs the
  gate with a recording command seam. A command that runs make or gmake with the target test, tests,
  spec, specs, check, ci or verify is a whole-project check and is skipped like a test suite; other
  make targets and the commands recognised before keep their answers.

  Scenario: Whole-project make targets are test suite checks
    Then shape-static-gate-skips-make-targets: each of these commands is a test suite check:
      """
      make spec
      make test
      make -j4 check
      cd app && make verify
      MIX_ENV=test make spec
      gmake ci
      make -C app tests
      """

  Scenario: Other make targets are not test suite checks
    Then shape-static-gate-skips-make-targets: none of these commands is a test suite check:
      """
      make lint
      make fmt
      make specimen
      make test-unit
      make -C spec lint
      """

  Scenario: The gate does not hand the make target to the command seam
    When shape-static-gate-skips-make-targets: the static gate runs the checks "spec" and "vet" with a recording command seam
    Then shape-static-gate-skips-make-targets: the only command given to the command seam is the "vet" check command
    And shape-static-gate-skips-make-targets: the check "spec" has the status "not_run"
    And shape-static-gate-skips-make-targets: the progress shows the line "kogen: static check spec not run: it runs the test suite"

  Scenario: The gate still judges the shaped files with the other check
    When shape-static-gate-skips-make-targets: the static gate runs the checks "spec" and "vet" with a recording command seam
    Then shape-static-gate-skips-make-targets: the check "vet" has the status "attributable"
    And shape-static-gate-skips-make-targets: the check "vet" names the shaped file
    And shape-static-gate-skips-make-targets: the gate outcome is "refused"

  Scenario: Previously recognised commands keep their answers
    Then shape-static-gate-skips-make-targets: each of these commands is a test suite check:
      """
      go test ./...
      cd app && npm test
      pytest -q
      bin/rails test
      bundle exec rake test
      make && (cargo test)
      """
    And shape-static-gate-skips-make-targets: none of these commands is a test suite check:
      """
      go build ./...
      go testdata
      bin/rails routes
      rake testdata
      """
