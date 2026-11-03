Feature: Shaping repairs an unfinished or rejected package before it must submit again
  After a refused submission, or one whose readiness failures came back for acknowledgment, the
  first tool-free answer is followed by one recovery inference that does not require submit and
  carries the concrete feedback. An unchanged package earns no recovery, and stops name the
  earliest unresolved refusal next to the final one. Scripted loopback provider, synthetic
  repositories, no model.

  Scenario: An Intent-only package is investigated and completed before the next submission
    When the repair journey "missing-feature" runs on the "responses" route
    Then the repair journey exits with code 0
    And the repair journey sent 6 shaper requests
    And the repair journey requests that required submit are "none"
    And the repair journey recovery requests are "3"
    And the repair journey recovery events are:
      | request | attempt | cause   | gate   |
      | 3       | 1       | refused | intent |
    And the repair journey feedback to call "c1" is:
      """
      invalid submission:
      acceptance.feature does not exist
      Earlier well-formed proposals are retained; submit only corrections.
      """
    And the repair journey recovery message of request 3 is:
      """
      Recovery before submitting again: submission attempt 1 was refused at gate intent and nothing was published. Your accumulated proposals are kept, as the cumulative proposal manifest lists. This inference does not require submit: investigate with read_file, list_files, search_text, probe or documentation, add or correct files with propose_files, then call submit with the repaired package. An unchanged package is refused again. The refusal:
      acceptance.feature does not exist
      """
    And the repair journey feedback to call "list" contains "features/steps/calc_steps.js"
    And the repair journey feedback to call "add" contains "retained: .kogen/intents/is-even/acceptance.feature, features/steps/is-even__steps.js; 3 proposed file(s) in all."
    And the repair journey feedback to call "c2" contains "Readiness observations"
    And the repair journey has no stop text
    And the repair journey published its final package

  Scenario: Without API tool forcing the unfinished package gets the same recovery
    When the repair journey "missing-feature" runs on the "anthropic" route
    Then the repair journey exits with code 0
    And the repair journey sent 6 shaper requests
    And the repair journey requests that required submit are "none"
    And the repair journey recovery requests are "3"
    And the repair journey recovery message of request 3 is:
      """
      Recovery before submitting again: submission attempt 1 was refused at gate intent and nothing was published. Your accumulated proposals are kept, as the cumulative proposal manifest lists. This inference does not require submit: investigate with read_file, list_files, search_text, probe or documentation, add or correct files with propose_files, then call submit with the repaired package. An unchanged package is refused again. The refusal:
      acceptance.feature does not exist
      """
    And the repair journey published its final package

  Scenario: Readiness failures can be investigated before they are acknowledged
    When the repair journey "observations-then-prose" runs on the "responses" route
    Then the repair journey exits with code 0
    And the repair journey sent 3 shaper requests
    And the repair journey requests that required submit are "none"
    And the repair journey recovery requests are "3"
    And the repair journey recovery events are:
      | request | attempt | cause              | gate     |
      | 3       | 1       | readiness_failures | (absent) |
    And the repair journey recovery message of request 3 is:
      """
      Recovery before submitting again: submission attempt 1 returned readiness failures for acknowledgment (see .kogen/local/shaping/is-even/observations-1.txt) and nothing was published. Your accumulated proposals are kept, as the cumulative proposal manifest lists. This inference does not require submit: investigate with read_file, list_files, search_text, probe or documentation, add or correct files with propose_files, then call submit with the repaired package, or acknowledge legitimate missing-feature red as the readiness feedback describes.
      """
    And the repair journey published its final package

  Scenario: An unprepared fresh-copy database is inspected and its preparation repaired
    When the repair journey "unprepared-database" runs on the "responses" route
    Then the repair journey exits with code 0
    And the repair journey sent 6 shaper requests
    And the repair journey requests that required submit are "none"
    And the repair journey recovery requests are "3"
    And the repair journey recovery events are:
      | request | attempt | cause   | gate            |
      | 3       | 1       | refused | contract_review |
    And the repair journey feedback to call "c1" contains "Environment check: 1 acceptance step(s) fail on the unchanged tree because Kogen's fresh copy is not prepared for them"
    And the repair journey feedback to call "c1" contains "- List active accounts | fresh-copy output: missing accounts database"
    And the repair journey recovery message of request 3 is:
      """
      Recovery before submitting again: submission attempt 1 was refused at gate contract_review and nothing was published. Your accumulated proposals are kept, as the cumulative proposal manifest lists. This inference does not require submit: investigate with read_file, list_files, search_text, probe or documentation, add or correct files with propose_files, then call submit with the repaired package. An unchanged package is refused again. The refusal:
      - List active accounts | fresh-copy output: missing accounts database | why: the database is prepared only in the shaping checkout | confirmed: the fresh copy has no prepared database
      """
    And the repair journey feedback to call "look" contains "console.error('missing accounts database')"
    And the repair journey feedback to call "try" contains "unknown option --active"
    And the repair journey feedback to call "c2" contains "unknown option --active"
    And the repair journey feedback to call "c2" does not contain "Environment check:"
    And the repair journey published its final package

  Scenario: The unchanged unprepared package stays refused and earns no second recovery
    When the repair journey "unprepared-database-unchanged" runs on the "responses" route
    Then the repair journey exits with code 1
    And the repair journey sent 5 shaper requests
    And the repair journey requests that required submit are "5"
    And the repair journey recovery requests are "3"
    And the repair journey feedback to call "c1" does not contain "Unchanged:"
    And the repair journey feedback to call "c2" ends with:
      """
      - List active accounts | fresh-copy output: missing accounts database | why: the database is prepared only in the shaping checkout | confirmed: the fresh copy has no prepared database
      Unchanged: this package is identical to the one refused at attempt 1; repair it before submitting again.
      """
    And the repair journey stop text is:
      """
      intent/not_ready: the Intent is not ready to build
        - List active accounts | fresh-copy output: missing accounts database | why: the database is prepared only in the shaping checkout | confirmed: the fresh copy has no prepared database
      """

  Scenario: An omitted requested property gets a scenario that exercises it
    When the repair journey "property-repaired" runs on the "responses" route
    Then the repair journey exits with code 0
    And the repair journey sent 6 shaper requests
    And the repair journey requests that required submit are "none"
    And the repair journey recovery requests are "3"
    And the repair journey feedback to call "c1" contains "- slug results are at most 20 characters (new) | request: Its result is at most 20 characters long. | problem: no scenario exercises it"
    And the repair journey feedback to call "look" contains "exports.slug = (text) => text;"
    And the repair journey feedback to call "c2" contains "Readiness observations"
    And the repair journey feedback to call "c2" does not contain "Property check:"
    And the repair journey published its final package

  Scenario: Weakening the property's assertion does not make the package ready
    When the repair journey "property-weakened" runs on the "responses" route
    Then the repair journey exits with code 1
    And the repair journey sent 7 shaper requests
    And the repair journey requests that required submit are "7"
    And the repair journey recovery requests are "3, 6"
    And the repair journey stop text is:
      """
      shape/invalid_output: the shaper answered without calling any tool 3 times since its last submission (no-tool limit); that submission (attempt 2) was refused at gate contract_review
        the shaper ended without calling submit
        last refusal (attempt 2, gate contract_review):
        - slug results are at most 20 characters (new) | request: Its result is at most 20 characters long. | problem: every scenario that exercises it passes on the unchanged tree, so none is red at an assertion about it | confirmed: the weakened step passes without the bound
        earliest unresolved refusal (attempt 1, gate contract_review):
        - slug results are at most 20 characters (new) | request: Its result is at most 20 characters long. | problem: no scenario exercises it | confirmed: no scenario covers the bound
        last tool-free answer: Done.
      """

  Scenario: Deleting the property from the Intent does not make the package ready
    When the repair journey "property-deleted" runs on the "responses" route
    Then the repair journey exits with code 1
    And the repair journey sent 5 shaper requests
    And the repair journey requests that required submit are "none"
    And the repair journey recovery requests are "3, 5"
    And the repair journey feedback to call "c2" contains "problem: no scenario exercises it | confirmed: the Request still asks for the bound"
    And the repair journey stop text is:
      """
      intent/not_ready: the Intent is not ready to build
        - slug results are at most 20 characters (new) | request: Its result is at most 20 characters long. | problem: no scenario exercises it | confirmed: the Request still asks for the bound
        earliest unresolved refusal (attempt 1, gate contract_review):
        - slug results are at most 20 characters (new) | request: Its result is at most 20 characters long. | problem: no scenario exercises it | confirmed: no scenario covers the bound
      """

  Scenario: A refused technical question and an incomplete package stop with both causes
    When the repair journey "question-then-incomplete" runs on the "responses" route
    Then the repair journey exits with code 1
    And the repair journey sent 5 shaper requests
    And the repair journey requests that required submit are "5"
    And the repair journey recovery requests are "4"
    And the repair journey feedback to call "c1" contains "- question 1 (technical): Which database library should store the accounts?"
    And the repair journey stop text is:
      """
      shape/invalid_output: the shaper answered without calling any tool 3 times since its last submission (no-tool limit); that submission (attempt 2) was refused at gate intent
        the shaper ended without calling submit
        last refusal (attempt 2, gate intent):
        acceptance.feature does not exist
        earliest unresolved refusal (attempt 1, gate question_check):
        - question 1 (technical): Which database library should store the accounts? | why: a technical question | decide it from what the repository and the request establish
        last tool-free answer: I cannot finish the package.
      """

  Scenario: Prose between refused submissions still ends after nine requests
    When the repair journey "prose-and-refusals" runs on the "responses" route
    Then the repair journey exits with code 1
    And the repair journey sent 9 shaper requests
    And the repair journey requests that required submit are "2, 3, 6, 9"
    And the repair journey recovery requests are "5, 8"
    And the repair journey stop text is:
      """
      shape/invalid_output: the shaper answered without calling any tool 3 times since its last submission (no-tool limit); that submission (attempt 2) was refused at gate validation
        the shaper ended without calling submit
        last refusal (attempt 2, gate validation):
        submit takes exactly one of a non-empty questions list, a non-empty files list, or a blocked report
        last tool-free answer: Last words.
      """

  Scenario: A valid first submission takes the ordinary ready path
    When the repair journey "valid-first" runs on the "responses" route
    Then the repair journey exits with code 0
    And the repair journey sent 1 shaper requests
    And the repair journey requests that required submit are "none"
    And the repair journey recovery requests are "none"
    And the repair journey has no stop text
    And the repair journey published its final package

  Scenario: An open interface choice asked during recovery still reaches the user
    When the repair journey "ui-question-after-refusal" runs on the "responses" route
    Then the repair journey exits with code 5
    And the repair journey sent 3 shaper requests
    And the repair journey requests that required submit are "none"
    And the repair journey recovery requests are "3"
    And the repair journey output contains "1. Should an empty list show a message or stay blank?"
