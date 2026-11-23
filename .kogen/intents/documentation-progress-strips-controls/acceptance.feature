Feature: Documentation progress references stay on one printable line
  This feature checks that the started progress line of a documentation lookup shows
  its reference with every control character turned into a space and ordinary text kept.

  Scenario: Carriage return, line feed, tab, NUL and ESC become spaces
    Given documentation-progress-strips-controls: a retained search lookup whose query is "cache{U+000D}{U+000A}{U+0009}{U+0000}{U+001B}controls"
    When documentation-progress-strips-controls: the documentation progress lines are produced
    Then documentation-progress-strips-controls: the started progress line is one line without control characters
    And documentation-progress-strips-controls: the started progress line is "kogen: documentation: search cache     controls"

  Scenario: Control characters in a fetched url become spaces
    Given documentation-progress-strips-controls: a retained fetch lookup whose url is "https://example.test/a{U+001B}[2Jb{U+007F}c"
    When documentation-progress-strips-controls: the documentation progress lines are produced
    Then documentation-progress-strips-controls: the started progress line is one line without control characters
    And documentation-progress-strips-controls: the started progress line is "kogen: documentation: fetch https://example.test/a [2Jb c"

  Scenario: Ordinary spaces and printable Unicode stay intact
    Given documentation-progress-strips-controls: a retained search lookup whose query is "café  €5 — 日本語 😀"
    When documentation-progress-strips-controls: the documentation progress lines are produced
    Then documentation-progress-strips-controls: the started progress line is "kogen: documentation: search café  €5 — 日本語 😀"

  Scenario: A line break in the reference still becomes one space
    Given documentation-progress-strips-controls: a retained search lookup whose query is "missing{U+000A}provider"
    When documentation-progress-strips-controls: the documentation progress lines are produced
    Then documentation-progress-strips-controls: the started progress line is "kogen: documentation: search missing provider"
