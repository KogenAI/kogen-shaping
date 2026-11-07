Feature: Tell the shaper the forms its gates require
  The shaper prompt names, before the first submission, every form Kogen's own gates require, and
  three gates accept exactly what the request asks them to: a marker anywhere on the line that
  holds a checked decision statement, an evidence item that starts with tracked paths followed by
  prose, and a bounded number of small files under the Intent's own folder. Every other case stays
  refused. Each case is a synthetic request shaped natively with a scripted shaper and a scripted
  decision-check model on the scripted-provider fixture project of shape/helper_test.go; the harness
  adds the tracked files tests/x_test.py and src/calc.py to it. The evidence items of the cases are
  `tests/x_test.py and the check runs it`, `src/missing.py`, `calc.add`, `see src/calc.py`,
  `tests/x_test.py and src/new.py` and `src/calc.py and the check runs it`, the last one on a
  shaping that never read src/calc.py.

  Scenario: The shaper prompt names every form the gates require before the first submission
    When k-shape-prompt-states-gate-forms: the "prompt" case is shaped
    Then k-shape-prompt-states-gate-forms: the instructions of the first shaper request contain these literals
      | literal |
      | (assumption: <what the user must confirm>) |
      | (evidence: <tracked path>) |
      | - <choice the repository settles> (evidence: <tracked path>) (source: request) |
      | - <choice decided without evidence> (assumption: <what the user must confirm>) (source: request) |
      | ran `<command>` |
      | Evidence items are paths and commands only, never prose. |
      | Markers go before the trailing tag: every Decisions line ends with exactly (source: request) or (source: answer <n>), and nothing follows it. |
      | A repository fact (test location, framework, check command, current or unchanged behavior) is either cited with an evidence marker or left out. |
      | Concrete examples (inputs and their results) belong in acceptance.feature, not in intent.md or Tests lines. |
      | - <rule>: <test> (<layer>) |
      | with the layer last |
      | Step sentences are Cucumber Expressions |
      | is optional text |
      | A step that checks an exit code runs a built program |
      | not go run |
      | Given steps pass on the unchanged tree; each scenario's failure today comes from a Then step. |
      | <Short prose: the outcome and its boundaries as the request states them, including what the request replaces or removes.> |
    And k-shape-prompt-states-gate-forms: the instructions of the first shaper request do not contain "including what existing behavior is replaced or removed"

  Scenario: The prompt goldens record the new prompt
    Then k-shape-prompt-states-gate-forms: these prompt golden tests of the shape package pass
      | test |
      | TestShaperPolicyGolden |
      | TestPromptsAreByteEqualToTheElixirFunctions |
      | TestShapeForceWireCompatibility |
    And k-shape-prompt-states-gate-forms: the prompt recorded in the policy golden contains these literals
      | literal |
      | (evidence: <tracked path>) |
      | Concrete examples (inputs and their results) belong in acceptance.feature, not in intent.md or Tests lines. |
      | Step sentences are Cucumber Expressions |
      | Given steps pass on the unchanged tree; each scenario's failure today comes from a Then step. |

  Scenario: A marker elsewhere on the statement's own line counts
    When k-shape-prompt-states-gate-forms: the "decision-marker-on-line" case is shaped
    Then k-shape-prompt-states-gate-forms: the number of decision check requests is 1
    And k-shape-prompt-states-gate-forms: no line of the feedback to the first submission starts with "Decision check:"
    And k-shape-prompt-states-gate-forms: the Intent is published from the first submission

  Scenario Outline: A statement without a marker on every line it occupies stays refused
    When k-shape-prompt-states-gate-forms: the "<case>" case is shaped
    Then k-shape-prompt-states-gate-forms: the first submission is refused by "Decision check: " with a line containing "<statement>" and "<verdict>"

    Examples:
      | case                                 | statement                                                                                   | verdict               |
      | decision-marker-next-line            | statement: Odd numbers return False                                                         | verdict: unsupported  |
      | decision-two-lines-one-marker        | statement: Negative numbers follow the same rule (evidence: README.md) and zero is even.    | verdict: unsupported  |
      | decision-fabricated-quote            | statement: - Odd numbers return False (source: request)                                     | verdict: unsupported  |
      | decision-contradiction-beside-marked | statement: - Inputs are limited to 10,000 (source: request)                                 | verdict: contradicts  |
      | decision-unmarked-unsupported        | statement: - Odd numbers return False (source: request)                                     | verdict: unsupported  |

  Scenario: An evidence item that starts with a tracked path and goes on in prose cites that path
    When k-shape-prompt-states-gate-forms: the "evidence-path-then-prose" case is shaped
    Then k-shape-prompt-states-gate-forms: no line of the feedback to the first submission starts with "Decision evidence:"
    And k-shape-prompt-states-gate-forms: the Intent is published from the first submission

  Scenario Outline: Evidence items that are not tracked, read paths followed by prose stay refused
    When k-shape-prompt-states-gate-forms: the "<case>" case is shaped
    Then k-shape-prompt-states-gate-forms: the first submission is refused by "Decision evidence: " with the line "<line>"

    Examples:
      | case                     | line                                                                    |
      | evidence-missing         | - decision 1: src/missing.py is not a file or directory tracked at HEAD  |
      | evidence-symbol          | - decision 1: calc.add is not a file or directory tracked at HEAD        |
      | evidence-see             | - decision 1: see src/calc.py is not a file or directory tracked at HEAD |
      | evidence-untracked-later | - decision 1: src/new.py is not a file or directory tracked at HEAD      |
      | evidence-unread          | - decision 1: this shaping did not read src/calc.py                      |

  Scenario: Shaping writes up to 16 small files under the Intent's own folder
    When k-shape-prompt-states-gate-forms: the "write-fixture" case is shaped
    Then k-shape-prompt-states-gate-forms: the Intent is published from the first submission
    And k-shape-prompt-states-gate-forms: every submitted file under the Intent folder is written with its exact content

  Scenario Outline: Files elsewhere, more files or larger files stay refused
    When k-shape-prompt-states-gate-forms: the "<case>" case is shaped
    Then k-shape-prompt-states-gate-forms: the first submission is refused by "invalid submission:" with a line starting "<start>" that contains "<text>"
    And k-shape-prompt-states-gate-forms: every refused file is unchanged by shaping

    Examples:
      | case                  | start                                                           | text                                                                |
      | write-product         | calc.js:                                                        | shaping may write only                                              |
      | write-project         | .kogen/project.yaml:                                            | shaping may write only                                              |
      | write-other-intent    | .kogen/intents/other/input.txt:                                 | shaping may write only                                              |
      | write-step-unprefixed | features/steps/helper.js:                                       | new step files must be named                                        |
      | write-steps-json      | features/steps/is-even__data.json:                              | new step files must be named                                        |
      | write-traversal       | .kogen/intents/is-even/../other/x.txt:                          | paths must be relative and normal                                   |
      | write-17-extra        | shaping may write at most 16 files under .kogen/intents/is-even/ | besides intent.md, acceptance.feature, model.qnt and controls.json |
      | write-oversize        | .kogen/intents/is-even/fixtures/big.txt:                        | may hold at most 65536 bytes                                        |
