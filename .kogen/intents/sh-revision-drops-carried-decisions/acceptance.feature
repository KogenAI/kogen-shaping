Feature: Restore carried Decisions during refused draft revision
  Scenario: Restore the eight named round-one decisions
    Given sh-revision-drops-carried-decisions: a refused draft and exact keep-lines
      """
      {"draft":"# Docs\n\n## Decisions\n- Keep task-first documentation. (source: request)\n\n## Tests\n- none: documentation\n","lines":["- Synthetic docs choice 9 (decision: D9, round 1) (source: request)","- Synthetic docs choice 10 (decision: D10, round 1) (source: request)","- Synthetic docs choice 11 (decision: D11, round 1) (source: request)","- Synthetic docs choice 12 (decision: D12, round 1) (source: request)","- Synthetic docs choice 13 (decision: D13, round 1) (source: request)","- Synthetic docs choice 14 (decision: D14, round 1) (source: request)","- Synthetic docs choice 15 (decision: D15, round 1) (source: request)","- Synthetic docs choice 16 (decision: D16, round 1) (source: request)"]}
      """
    Then sh-revision-drops-carried-decisions: revising twice produces exactly
      """
      # Docs

      ## Decisions
      - Keep task-first documentation. (source: request)
      - Synthetic docs choice 9 (decision: D9, round 1) (source: request)
      - Synthetic docs choice 10 (decision: D10, round 1) (source: request)
      - Synthetic docs choice 11 (decision: D11, round 1) (source: request)
      - Synthetic docs choice 12 (decision: D12, round 1) (source: request)
      - Synthetic docs choice 13 (decision: D13, round 1) (source: request)
      - Synthetic docs choice 14 (decision: D14, round 1) (source: request)
      - Synthetic docs choice 15 (decision: D15, round 1) (source: request)
      - Synthetic docs choice 16 (decision: D16, round 1) (source: request)

      ## Tests
      - none: documentation
      """

  Scenario: Preserve existing exact decisions and their metadata
    Given sh-revision-drops-carried-decisions: a refused draft and exact keep-lines
      """
      {"draft":"# Docs\n\n## Decisions\n- Keep concise pages. (decision: D8, round 1) (constraint: task first) (evidence: README.md) (source: request)\n\n## Tests\n- none: documentation\n","lines":["- Keep concise pages. (decision: D8, round 1) (constraint: task first) (evidence: README.md) (source: request)","- Keep examples. (decision: D9, round 1) (constraint: retain commands) (evidence: README.md) (source: request)"]}
      """
    Then sh-revision-drops-carried-decisions: revising twice produces exactly
      """
      # Docs

      ## Decisions
      - Keep concise pages. (decision: D8, round 1) (constraint: task first) (evidence: README.md) (source: request)
      - Keep examples. (decision: D9, round 1) (constraint: retain commands) (evidence: README.md) (source: request)

      ## Tests
      - none: documentation
      """
