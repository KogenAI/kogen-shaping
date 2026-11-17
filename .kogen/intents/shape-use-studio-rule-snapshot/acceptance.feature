Feature: Shape uses the Studio rule snapshot
  This feature drives the real shape role entry with the shape tests' scripted provider, which answers every model call, and a local fake authenticated Studio endpoint with a synthetic token.

  Scenario: Live rules reach the model request and the receipt
    Given shape-use-studio-rule-snapshot: the fake Studio endpoint serves rule rule-repo-1 at revision 3 for the synthetic token
    When shape-use-studio-rule-snapshot: the live journey is run
    Then shape-use-studio-rule-snapshot: the live run succeeds, every shaper and contract-review request carries rule-repo-1 revision 3, and the receipt records rule-repo-1 revision 3 with its snapshot hash and origin live

  Scenario: A changed rule revision reaches the next run
    Given shape-use-studio-rule-snapshot: the fake Studio endpoint serves rule rule-repo-1 at revision 3 for the synthetic token
    When shape-use-studio-rule-snapshot: the changed journey is run
    Then shape-use-studio-rule-snapshot: the second run carries and records rule-repo-1 revision 4 with no prompt file edited

  Scenario: An unconfigured project keeps standalone behavior
    Given shape-use-studio-rule-snapshot: a project with no Studio configuration
    When shape-use-studio-rule-snapshot: the standalone journey is run
    Then shape-use-studio-rule-snapshot: the run succeeds and no rule receipt entry is written

  Scenario: The answer run records the rules of its requests
    Given shape-use-studio-rule-snapshot: the fake Studio endpoint serves rule rule-repo-1 at revision 3 for the synthetic token
    When shape-use-studio-rule-snapshot: the answer journey is run
    Then shape-use-studio-rule-snapshot: the answer run's requests carry rule-repo-1 revision 3 and its answer boundary records rule-repo-1 revision 3, the snapshot hash and origin live

  Scenario: An outage uses the last verified snapshot
    Given shape-use-studio-rule-snapshot: the fake Studio endpoint serves rule rule-repo-1 at revision 3 for the synthetic token
    When shape-use-studio-rule-snapshot: the outage journey is run
    Then shape-use-studio-rule-snapshot: the outage run's receipt says origin cached with rule-repo-1 revision 3 and its requests carry revision 3 only

  Scenario: Revoked, foreign and uncached cases are named and never reuse a cache
    Given shape-use-studio-rule-snapshot: the fake Studio endpoint serves rule rule-repo-1 at revision 3 for the synthetic token
    When shape-use-studio-rule-snapshot: the denied journey is run
    Then shape-use-studio-rule-snapshot: the revoked token run is refused as rules/refused http_401 and no rule set reaches a request
    When shape-use-studio-rule-snapshot: the foreign journey is run
    Then shape-use-studio-rule-snapshot: the foreign scope run is refused as rules/refused http_403 and no rule set reaches a request
    When shape-use-studio-rule-snapshot: the foreign_down journey is run
    Then shape-use-studio-rule-snapshot: the other scope during an outage is reported as rules/unavailable http_503 and the first scope cache is not reused
    When shape-use-studio-rule-snapshot: the cold_down journey is run
    Then shape-use-studio-rule-snapshot: the unavailable run without a cache is reported as rules/unavailable http_503 and no empty rule set is used
