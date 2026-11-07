Feature: Name late gate forms before the first submission
  Scenario: Decision form is supplied in advance
    Then k-shape-decision-lines-and-controls-named: shaper guidance contains these literals
      | literal |
      | D<n> |
      | (decision: D<n>, round <r>) |
      | (constraint: <c>) |
      | (assumption: <uncertainty>) |
      | (revises: <id> because <why>) |
      | (source: request) |
      | HEAD |

  Scenario: Controls form is supplied in advance
    Then k-shape-decision-lines-and-controls-named: shaper guidance contains these literals
      | literal |
      | .kogen/intents/<slug>/controls.json |
      | "correct" |
      | "wrong" |
      | "files" |
      | "scenario" |
      | "step" |
      | keyword included |
      | 65536 |

  Scenario: External sessions are told their commands are not decision evidence
    Then k-shape-decision-lines-and-controls-named: external guidance contains these literals
      | literal |
      | decision |
      | evidence |
      | uncertainty |

  Scenario: A refusal teaches the exact keyword form
    Then k-shape-decision-lines-and-controls-named: controls diagnostic contains "write the step line as acceptance.feature writes it, keyword included:"
