Feature: The static gate skips Rails test suite commands
  This feature asks the static gate's suite recognition about configured command lines. A command
  that runs a Rails test suite is a test suite check, which the gate never runs. Commands that only
  look similar, and the commands recognised before, keep their answers.

  Scenario: Rails suite commands are test suite checks
    Then shape-static-gate-skips-rails-suite: each of these commands is a test suite check:
      """
      bin/rails test
      rails test test/models
      bundle exec rake test
      RAILS_ENV=test bin/rake test:system
      cd app && bin/rails test
      """

  Scenario: Other Rails and rake commands are not test suite checks
    Then shape-static-gate-skips-rails-suite: none of these commands is a test suite check:
      """
      bin/rails routes
      bin/rubocop
      rake testdata
      echo rails testing
      """

  Scenario: Previously recognised commands keep their answers
    Then shape-static-gate-skips-rails-suite: each of these commands is a test suite check:
      """
      go test ./...
      cd app && npm test
      pytest -q
      """
    And shape-static-gate-skips-rails-suite: none of these commands is a test suite check:
      """
      go build ./...
      go testdata
      """
