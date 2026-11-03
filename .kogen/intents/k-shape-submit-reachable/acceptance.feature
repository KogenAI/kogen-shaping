Feature: Shaping reaches a valid submit after prose answers and refusals
  A native Shape conversation counts tool-free answers since the shaper's last submission, ends at
  the no-tool limit with the cause the shaper was asked to fix, and records the text of every
  tool-free answer in its events. Scripted loopback provider, synthetic repository, no model.

  Scenario: Prose and refused submissions alternate, then the forced inference submits
    When Shape runs the scripted "prose-refusal-cycle" conversation on the "responses" route
    Then Shape exits with code 0
    And the provider received 6 shaping requests
    And the requests that required submit are "2, 4, 6"
    And there is no stop text
    And the Intent file intent.md is written

  Scenario: A shaper that only answers in prose stops with its last answer
    When Shape runs the scripted "prose-only" conversation on the "responses" route
    Then Shape exits with code 1
    And the provider received 3 shaping requests
    And the requests that required submit are "2, 3"
    And no Intent file intent.md is written
    And the stop text is:
      """
      shape/invalid_output: the shaper answered without calling any tool 3 times (no-tool limit); no submission was made
        the shaper ended without calling submit
        last tool-free answer: I need to know the database first.
      """

  Scenario: The stop names the gate and problem of the last refusal
    When Shape runs the scripted "refusal-then-prose" conversation on the "responses" route
    Then Shape exits with code 1
    And the provider received 5 shaping requests
    And the requests that required submit are "2, 4, 5"
    And no Intent file intent.md is written
    And the stop text is:
      """
      shape/invalid_output: the shaper answered without calling any tool 3 times since its last submission (no-tool limit); that submission (attempt 1) was refused at gate validation
        the shaper ended without calling submit
        last refusal (attempt 1, gate validation):
        submit takes exactly one of a non-empty questions list, a non-empty files list, or a blocked report
        last tool-free answer: Waiting for the readiness result.
      """

  Scenario: The stop names readiness failures left unacknowledged
    When Shape runs the scripted "observations-then-prose" conversation on the "responses" route
    Then Shape exits with code 1
    And the provider received 4 shaping requests
    And the requests that required submit are "3, 4"
    And the stop text is:
      """
      shape/invalid_output: the shaper answered without calling any tool 3 times since its last submission (no-tool limit); that submission (attempt 1) returned readiness failures that were not acknowledged
        the shaper ended without calling submit
        last submission (attempt 1): readiness failures returned for acknowledgment, see .kogen/local/shaping/is-even/observations-1.txt
        last tool-free answer: Done here.
      """

  Scenario: Prose before a valid submit keeps the normal outcome and records the prose
    When Shape runs the scripted "normal" conversation on the "responses" route
    Then Shape exits with code 0
    And the provider received 3 shaping requests
    And the requests that required submit are "3"
    And there is no stop text
    And the Intent file intent.md is written
    And the tool-free answer events are:
      | request | no_tool_turns | chars | truncated | text_runes | text_start         |
      | 2       | 1             | 18    | false     | 18         | Thinking out loud. |

  Scenario: The Chat Completions route records the prose answer too
    When Shape runs the scripted "normal" conversation on the "chat" route
    Then Shape exits with code 0
    And the provider received 3 shaping requests
    And the requests that required submit are "3"
    And the Intent file intent.md is written
    And the tool-free answer events are:
      | request | no_tool_turns | chars | truncated | text_runes | text_start         |
      | 2       | 1             | 18    | false     | 18         | Thinking out loud. |

  Scenario: Tool-free answer text is redacted, bounded at 2000 characters and absent when empty
    When Shape runs the scripted "answer-text" conversation on the "responses" route
    Then Shape exits with code 0
    And the provider received 5 shaping requests
    And the requests that required submit are "2, 3, 5"
    And the Intent file intent.md is written
    And the tool-free answer events are:
      | request | no_tool_turns | chars | truncated | text_runes | text_start                               |
      | 1       | 1             | 39    | false     | 39         | Plan: token=[REDACTED] and then submit.  |
      | 2       | 2             | 0     | false     | 0          | (absent)                                 |
      | 4       | 3             | 2513  | true      | 2000       | Long answer: abcdefghijabcdefghijabcdefg |

  Scenario: Without API tool forcing a prose-only shaper still gets the actionable stop
    When Shape runs the scripted "prose-only" conversation on the "anthropic" route
    Then Shape exits with code 1
    And the provider received 3 shaping requests
    And the requests that required submit are "none"
    And the stop text is:
      """
      shape/invalid_output: the shaper answered without calling any tool 3 times (no-tool limit); no submission was made
        the shaper ended without calling submit
        last tool-free answer: I need to know the database first.
      """

  Scenario: Without API tool forcing the stop still names the last refusal
    When Shape runs the scripted "refusal-then-prose" conversation on the "anthropic" route
    Then Shape exits with code 1
    And the provider received 5 shaping requests
    And the requests that required submit are "none"
    And the stop text is:
      """
      shape/invalid_output: the shaper answered without calling any tool 3 times since its last submission (no-tool limit); that submission (attempt 1) was refused at gate validation
        the shaper ended without calling submit
        last refusal (attempt 1, gate validation):
        submit takes exactly one of a non-empty questions list, a non-empty files list, or a blocked report
        last tool-free answer: Waiting for the readiness result.
      """

  Scenario: Prose answers between refused submissions still end after a finite number of inferences
    When Shape runs the scripted "finite-bound" conversation on the "responses" route
    Then Shape exits with code 1
    And the provider received 9 shaping requests
    And the requests that required submit are "2, 3, 5, 6, 8, 9"
    And no Intent file intent.md is written
    And the stop text is:
      """
      shape/invalid_output: the shaper answered without calling any tool 3 times since its last submission (no-tool limit); that submission (attempt 2) was refused at gate validation
        the shaper ended without calling submit
        last refusal (attempt 2, gate validation):
        submit takes exactly one of a non-empty questions list, a non-empty files list, or a blocked report
        last tool-free answer: Last words.
      """
