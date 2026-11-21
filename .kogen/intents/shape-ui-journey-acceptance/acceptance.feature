Feature: Shape specifies usable journeys for new UI surfaces
  This feature runs Shape on a synthetic fixture and reads the instructions it sends.

  Scenario: A contributor profile request yields a journey from an existing entry point
    When shape-ui-journey-acceptance: Shape processes "Add a public contributor profile page."
    Then shape-ui-journey-acceptance: the shaper instructions state "The representative executable journey starts from an existing entry point of the product, as the actual actor the request names, and goes through to the requested visible result."
    Then shape-ui-journey-acceptance: the shaper instructions state "use the existing browser or end-to-end tooling"
    Then shape-ui-journey-acceptance: the shaper instructions state "A direct URL visit or a self-menu-only path does not satisfy the journey"
    Then shape-ui-journey-acceptance: the contract review instructions state "A direct URL visit or a self-menu-only path does not satisfy the journey"

  Scenario: A known public comment is visible as its exact entry
    When shape-ui-journey-acceptance: Shape processes "Show a contributor's public comments on a Comments tab."
    Then shape-ui-journey-acceptance: the shaper instructions state "A count, a placeholder or a heading alone does not show the visible result"
    Then shape-ui-journey-acceptance: the shaper instructions state "an exact entry the fixtures define must be visible"

  Scenario: Conventions settle defaults and non-UI work gets no mandate
    When shape-ui-journey-acceptance: Shape processes "Add a public contributor profile page."
    Then shape-ui-journey-acceptance: the shaper instructions state "ask no question it settles"
    Then shape-ui-journey-acceptance: the shaper instructions state "Ask the caller only when an unresolved choice materially changes the UX"
    Then shape-ui-journey-acceptance: the shaper instructions state "get no browser-test mandate"
