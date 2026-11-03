Feature: Score Shape without the assembled Kogen CLI
  Scenario Outline: Publish and report the Intent through the standalone command
    When I run standalone Shape on a local scoring checkout using "<source>"
    Then standalone Shape reports exit 0, the exact written Intent, and no stale Intent on exit 5

    Examples:
      | source |
      | file   |
      | stdin  |
