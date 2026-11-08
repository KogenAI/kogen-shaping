Feature: Decide recommended answers without questioning the caller
  Scenario: A recommended product answer becomes an assumption
    When k-shape-decide-not-ask-recommended: I exercise "recommended"
    Then k-shape-decide-not-ask-recommended: the observation is
      | exit       | 0     |
      | asked      | 0     |
      | assumption | true  |
      | pending    | false |
      | ready      | true  |

  Scenario: A product question without a recommendation still reaches the caller
    When k-shape-decide-not-ask-recommended: I exercise "unrecommended"
    Then k-shape-decide-not-ask-recommended: the observation is
      | exit    | 5    |
      | asked   | 1    |
      | pending | true |

  Scenario: A failed session cannot carry a stale question artifact into the next session
    When k-shape-decide-not-ask-recommended: I exercise "stale"
    Then k-shape-decide-not-ask-recommended: the observation is
      | failed         | true  |
      | second_started | true  |
      | carried        | false |

  Scenario Outline: Decision checks accept settled sources
    When k-shape-decide-not-ask-recommended: I exercise "<source>"
    Then k-shape-decide-not-ask-recommended: the observation is
      | accepted | true |
    Examples:
      | source     |
      | answer     |
      | assumption |
