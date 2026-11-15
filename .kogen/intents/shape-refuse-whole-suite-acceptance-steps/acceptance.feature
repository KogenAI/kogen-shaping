Feature: Refuse whole-suite acceptance steps within the approval budget
  This feature preserves the request from Studio issue kogen/e05fb12c-082a-415f-973c-22b1bad2ce86.
  Observed: usage-limit-packaging-isolation-abort-under-load ran the step
  'I run two uncached full Go suites concurrently' for 114 s; acceptance was
  stopped at 120 s and the Build was recorded intent/not_ready.
  k-shape-harness-clean-env-lifecycle stopped with environment/step_timeout at 120 s.
  Readiness on the unchanged tree can fail fast and hide the implemented runtime.
  These are historical reports, not a claim that this fixture reproduces their timing.
  The executable reproduction submits the concurrent command SOURCE without running suites.
  Shape must refuse whole-suite step code, allow targeted commands, state the
  120 s per-step approval limit in its contract, and leave other static checks unchanged.

  Background:
    Given shape-refuse-whole-suite-acceptance-steps: the independent Shape harness is compiled

  Scenario: Refuse concurrent uncached Go suites
    When shape-refuse-whole-suite-acceptance-steps: Shape inspects this submitted command
      """
      {"Sentence":"I run two uncached full Go suites concurrently","Body":"require('node:child_process').spawn('go', ['test', '-count=1', './...']); require('node:child_process').spawn('go', ['test', '-count=1', './...']);","Outcome":"refused","Runner":"go test"}
      """
    Then shape-refuse-whole-suite-acceptance-steps: the static outcome matches without executing the command

  Scenario: Refuse a shell whole Go suite
    When shape-refuse-whole-suite-acceptance-steps: Shape inspects this submitted command
      """
      {"Sentence":"I run the full Go suite","Body":"require('node:child_process').execSync('go test ./...');","Outcome":"refused","Runner":"go test"}
      """
    Then shape-refuse-whole-suite-acceptance-steps: the static outcome matches without executing the command

  Scenario: Refuse Elixir tests with no file
    When shape-refuse-whole-suite-acceptance-steps: Shape inspects this submitted command
      """
      {"Sentence":"I run all Elixir tests","Body":"require('node:child_process').execSync('mix test');","Outcome":"refused","Runner":"mix test"}
      """
    Then shape-refuse-whole-suite-acceptance-steps: the static outcome matches without executing the command

  Scenario: Allow one Go package
    When shape-refuse-whole-suite-acceptance-steps: Shape inspects this submitted command
      """
      {"Sentence":"I run one Go package","Body":"require('node:child_process').execSync('go test ./shape');","Outcome":"allowed"}
      """
    Then shape-refuse-whole-suite-acceptance-steps: the static outcome matches without executing the command

  Scenario: Allow a named Go test
    When shape-refuse-whole-suite-acceptance-steps: Shape inspects this submitted command
      """
      {"Sentence":"I run one named Go test","Body":"require('node:child_process').spawnSync('go', ['test', '-run', '^TestOne$', './...']);","Outcome":"allowed"}
      """
    Then shape-refuse-whole-suite-acceptance-steps: the static outcome matches without executing the command

  Scenario: Allow one Elixir test file
    When shape-refuse-whole-suite-acceptance-steps: Shape inspects this submitted command
      """
      {"Sentence":"I run one Elixir file","Body":"require('node:child_process').execSync('mix test test/one_test.exs');","Outcome":"allowed"}
      """
    Then shape-refuse-whole-suite-acceptance-steps: the static outcome matches without executing the command

  Scenario: State the approval runtime contract
    When shape-refuse-whole-suite-acceptance-steps: Shape inspects this submitted command
      """
      {"Sentence":"I run one Go package","Body":"require('node:child_process').execSync('go test ./shape');","Outcome":"allowed"}
      """
    Then shape-refuse-whole-suite-acceptance-steps: the static outcome matches without executing the command
    And shape-refuse-whole-suite-acceptance-steps: the rendered shaper contract contains these instructions
      """
      ["120 s per-step approval limit", "Refuse acceptance steps that cannot finish within that limit", "readiness on the unchanged tree can fail fast", "single test file, package or named test", "whole test suite"]
      """
