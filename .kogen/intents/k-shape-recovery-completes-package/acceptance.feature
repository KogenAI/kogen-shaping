Feature: A recovery turn completes the refused package before it counts
  Scripted native Shape journeys (loopback provider, no model) in the synthetic is-even
  repository. The Intent declares .kogen/intents/is-even/acceptance.feature and
  features/steps/is-even__steps.js; the replayed acceptance report is green.

  Scenario: The refusal names the missing step file and the step written in recovery is accepted
    When k-shape-recovery-completes-package: the scripted shaper runs the journey "missing-step-written"
    Then k-shape-recovery-completes-package: shaping exits with code 0
    And k-shape-recovery-completes-package: the shaper sent 3 model requests
    And k-shape-recovery-completes-package: the feedback to call "c1" is:
      """
      invalid submission:
      acceptance: features/steps/is-even__steps.js does not exist
      Earlier well-formed proposals are retained; submit only corrections.
      """
    And k-shape-recovery-completes-package: the refused attempts and gates are "1 intent"
    And k-shape-recovery-completes-package: the pre-checks follow attempts "none"
    And k-shape-recovery-completes-package: the invalid submissions counted are 1
    And k-shape-recovery-completes-package: there is no stop text
    And k-shape-recovery-completes-package: the complete package was published

  Scenario: After prose in recovery the unchanged package gets its remaining gap without spending an attempt
    When k-shape-recovery-completes-package: the scripted shaper runs the journey "missing-step-prose"
    Then k-shape-recovery-completes-package: shaping exits with code 0
    And k-shape-recovery-completes-package: the shaper sent 5 model requests
    And k-shape-recovery-completes-package: the requests carrying a recovery message are "3"
    And k-shape-recovery-completes-package: the feedback to call "c2" is:
      """
      not submitted: this is the same refusal as attempt 1; these items are unchanged, and this call was not counted as a submission attempt:
      acceptance: features/steps/is-even__steps.js does not exist
      The files of this call were retained with your earlier proposals. Add what is missing with propose_files; the next submission counts as an attempt.
      """
    And k-shape-recovery-completes-package: the refused attempts and gates are "1 intent"
    And k-shape-recovery-completes-package: the pre-checks follow attempts "1"
    And k-shape-recovery-completes-package: the gaps of pre-check 1 are:
      """
      acceptance: features/steps/is-even__steps.js does not exist
      """
    And k-shape-recovery-completes-package: the invalid submissions counted are 1
    And k-shape-recovery-completes-package: there is no stop text
    And k-shape-recovery-completes-package: the complete package was published

  Scenario: Every missing declared file is named and the pre-check returns the gap that remains
    When k-shape-recovery-completes-package: the scripted shaper runs the journey "intent-only"
    Then k-shape-recovery-completes-package: shaping exits with code 0
    And k-shape-recovery-completes-package: the shaper sent 3 model requests
    And k-shape-recovery-completes-package: the feedback to call "c1" is:
      """
      invalid submission:
      acceptance.feature does not exist
      acceptance: features/steps/is-even__steps.js does not exist
      Earlier well-formed proposals are retained; submit only corrections.
      """
    And k-shape-recovery-completes-package: the problems of refused attempt 1 are:
      """
      acceptance.feature does not exist
      acceptance: features/steps/is-even__steps.js does not exist
      """
    And k-shape-recovery-completes-package: the feedback to call "c2" is:
      """
      not submitted: the package still has these gaps, and this call was not counted as a submission attempt:
      acceptance: features/steps/is-even__steps.js does not exist
      The files of this call were retained with your earlier proposals. Add what is missing with propose_files; the next submission counts as an attempt.
      """
    And k-shape-recovery-completes-package: the refused attempts and gates are "1 intent"
    And k-shape-recovery-completes-package: the pre-checks follow attempts "1"
    And k-shape-recovery-completes-package: the invalid submissions counted are 1
    And k-shape-recovery-completes-package: there is no stop text
    And k-shape-recovery-completes-package: the complete package was published

  Scenario: A malformed envelope names the rule and the received shape and the repaired files are accepted
    When k-shape-recovery-completes-package: the scripted shaper runs the journey "envelope-repaired"
    Then k-shape-recovery-completes-package: shaping exits with code 0
    And k-shape-recovery-completes-package: the shaper sent 2 model requests
    And k-shape-recovery-completes-package: the feedback to call "c1" is:
      """
      invalid submission:
      submit takes exactly one of a non-empty questions list, a non-empty files list, or a blocked report
      received: questions (list of 1), files (list of 3)
      No proposals from this submission were retained; earlier proposals are unchanged.
      """
    And k-shape-recovery-completes-package: the refused attempts and gates are "1 validation"
    And k-shape-recovery-completes-package: the problems of refused attempt 1 are:
      """
      submit takes exactly one of a non-empty questions list, a non-empty files list, or a blocked report
      received: questions (list of 1), files (list of 3)
      """
    And k-shape-recovery-completes-package: the pre-checks follow attempts "none"
    And k-shape-recovery-completes-package: the invalid submissions counted are 1
    And k-shape-recovery-completes-package: there is no stop text
    And k-shape-recovery-completes-package: the complete package was published

  Scenario: A repeated malformed envelope is reported unchanged and ends with a blocked report at the budget
    When k-shape-recovery-completes-package: the scripted shaper runs the journey "envelope-repeated"
    Then k-shape-recovery-completes-package: shaping exits with code 1
    And k-shape-recovery-completes-package: the shaper sent 6 model requests
    And k-shape-recovery-completes-package: the requests carrying a recovery message are "3"
    And k-shape-recovery-completes-package: the feedback to call "c2" is:
      """
      not submitted: this is the same refusal as attempt 1; these items are unchanged, and this call was not counted as a submission attempt:
      submit takes exactly one of a non-empty questions list, a non-empty files list, or a blocked report
      received: questions (list of 1), files (list of 3)
      No proposals from this call were retained; earlier proposals are unchanged. The next submission counts as an attempt.
      """
    And k-shape-recovery-completes-package: the feedback to call "c3" is:
      """
      invalid submission:
      This is the same refusal as attempt 1; these items are unchanged:
      submit takes exactly one of a non-empty questions list, a non-empty files list, or a blocked report
      received: questions (list of 1), files (list of 3)
      No proposals from this submission were retained; earlier proposals are unchanged.
      """
    And k-shape-recovery-completes-package: the feedback to call "c4" is:
      """
      not submitted: this is the same refusal as attempt 2; these items are unchanged, and this call was not counted as a submission attempt:
      submit takes exactly one of a non-empty questions list, a non-empty files list, or a blocked report
      received: questions (list of 1), files (list of 3)
      No proposals from this call were retained; earlier proposals are unchanged. The next submission counts as an attempt.
      """
    And k-shape-recovery-completes-package: the refused attempts and gates are "1 validation, 2 validation, 3 validation"
    And k-shape-recovery-completes-package: the pre-checks follow attempts "1, 2"
    And k-shape-recovery-completes-package: the gaps of pre-check 2 are:
      """
      submit takes exactly one of a non-empty questions list, a non-empty files list, or a blocked report
      received: questions (list of 1), files (list of 3)
      """
    And k-shape-recovery-completes-package: the invalid submissions counted are 3
    And k-shape-recovery-completes-package: the stop text is:
      """
      shape/incomplete_package: blocked: the shaper could not complete the package within 3 attempts
        unresolved gap (attempt 3, gate validation):
        submit takes exactly one of a non-empty questions list, a non-empty files list, or a blocked report
        received: questions (list of 1), files (list of 3)
      """
    And k-shape-recovery-completes-package: no Intent was written

  Scenario: A complete first submission is accepted with no recovery and no extra request
    When k-shape-recovery-completes-package: the scripted shaper runs the journey "valid-first"
    Then k-shape-recovery-completes-package: shaping exits with code 0
    And k-shape-recovery-completes-package: the shaper sent 1 model requests
    And k-shape-recovery-completes-package: the requests carrying a recovery message are "none"
    And k-shape-recovery-completes-package: the refused attempts and gates are "none"
    And k-shape-recovery-completes-package: the pre-checks follow attempts "none"
    And k-shape-recovery-completes-package: the invalid submissions counted are 0
    And k-shape-recovery-completes-package: there is no stop text
    And k-shape-recovery-completes-package: the complete package was published
