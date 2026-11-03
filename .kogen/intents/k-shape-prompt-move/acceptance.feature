Feature: Keep Shaper policy bytes while moving their ownership
  Scenario: Compare the locally owned policy with the current executor pin
    When I compare the shaping-owned prompt and tools with the pinned executor
    Then the rendered prompt and ordered tool definitions are byte-identical
