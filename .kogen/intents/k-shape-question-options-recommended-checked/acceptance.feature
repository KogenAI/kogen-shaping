Feature: Every question shaping asks carries at least two distinct options and exactly one recommended option

  The harness features/steps/k-shape-question-options-recommended-checked__harness.js runs Shape (kogen intent shape)
  with the native shaper against the shape package's scripted local provider in the shape tests' invented is-even project.
  The scripted shaper submits the questions written in each case; the scripted question check classes every question it
  is shown ui_ux (handoff in the handoff cases). A case with an answer runs Shape a second time with that answer, after
  which the scripted shaper submits the shape tests' is-even package and readiness is green. The observations are what
  Shape printed (stdout), stored as pending (questions.json data and the question rounds of state.json, read after the
  first run), sent to the provider (request_kinds, in order) and returned to the shaper (feedback_<call id>). The default
  limit of 3 submissions applies.

  Scenario: A question with no options is refused with feedback, nothing is printed or stored, and the repaired question is asked once and answered by all: recommended
    When k-shape-question-options-recommended-checked: the "no-options" case is shaped
    Then k-shape-question-options-recommended-checked: the observations include:
      """
      {"exit": 5,
       "error": "",
       "refused_gates": ["validation"],
       "request_kinds": ["shaper", "shaper", "question_check", "shaper", "decision_check", "literal_check", "contract_check"],
       "questions_json": {"slug": "is-even", "questions": [
         {"number": 1, "question": "How should calc.is_even(n) answer a number that is not an integer?", "kind": "product", "recommended": "a",
          "options": [{"letter": "a", "label": "Raise TypeError", "recommended": true}, {"letter": "b", "label": "Return false", "recommended": false}]}]},
       "state_rounds": [[{"question": "How should calc.is_even(n) answer a number that is not an integer?",
                          "options": [{"label": "Raise TypeError", "recommended": true}, {"label": "Return false"}]}]],
       "answer_exit": 0,
       "answer_error": "",
       "answer_last_stdout_line": "Next: kogen intent approve is-even",
       "receipt_answers": {"1": "a"},
       "receipt_resolved_answers": {"1": "Raise TypeError"},
       "pending_after_answer": false,
       "published": true}
      """
    And k-shape-question-options-recommended-checked: the observation "feedback_q1" is exactly:
      """
      invalid submission:
      question 1 (How should calc.is_even(n) answer a number that is not an integer?): no options
      every question needs at least two options with distinct, non-empty labels and exactly one option marked recommended
      No proposals from this submission were retained; earlier proposals are unchanged.
      """
    And k-shape-question-options-recommended-checked: the observation "stdout" is exactly these lines:
      """
      Questions for is-even:
      1. How should calc.is_even(n) answer a number that is not an integer?
         a) Raise TypeError (recommended)
         b) Return false
      Answer one per line as <n>: <letter or your words>, e.g. printf '1: a\n' | kogen intent shape is-even -
      Or enter all: recommended to use every question's recommended option.
      """
    And k-shape-question-options-recommended-checked: the observation "question_check_input_1" is exactly:
      """
      Slug: is-even

      # Request

      Add calc.is_even(n), which returns true for even integers.


      # Earlier questions and answers

      none

      # Questions

      1. How should calc.is_even(n) answer a number that is not an integer?
         a) Raise TypeError (recommended)
         b) Return false
      """
    And k-shape-question-options-recommended-checked: the observation "earlier_round" is exactly:
      """
      Round 1:
      1. How should calc.is_even(n) answer a number that is not an integer?
         a) Raise TypeError
         b) Return false
         answer: 1: a
      """

  Scenario: A question with an empty option list is refused with feedback, and the repaired question is asked once and answered by all: recommended
    When k-shape-question-options-recommended-checked: the "empty-options" case is shaped
    Then k-shape-question-options-recommended-checked: the observations include:
      """
      {"exit": 5,
       "error": "",
       "refused_gates": ["validation"],
       "request_kinds": ["shaper", "shaper", "question_check", "shaper", "decision_check", "literal_check", "contract_check"],
       "questions_json": {"slug": "is-even", "questions": [
         {"number": 1, "question": "How should calc.is_even(n) answer a number that is not an integer?", "kind": "product", "recommended": "a",
          "options": [{"letter": "a", "label": "Raise TypeError", "recommended": true}, {"letter": "b", "label": "Return false", "recommended": false}]}]},
       "state_rounds": [[{"question": "How should calc.is_even(n) answer a number that is not an integer?",
                          "options": [{"label": "Raise TypeError", "recommended": true}, {"label": "Return false"}]}]],
       "answer_exit": 0,
       "answer_error": "",
       "receipt_answers": {"1": "a"},
       "receipt_resolved_answers": {"1": "Raise TypeError"},
       "published": true}
      """
    And k-shape-question-options-recommended-checked: the observation "feedback_q1" is exactly:
      """
      invalid submission:
      question 1 (How should calc.is_even(n) answer a number that is not an integer?): an empty option list
      every question needs at least two options with distinct, non-empty labels and exactly one option marked recommended
      No proposals from this submission were retained; earlier proposals are unchanged.
      """
    And k-shape-question-options-recommended-checked: the observation "stdout" is exactly these lines:
      """
      Questions for is-even:
      1. How should calc.is_even(n) answer a number that is not an integer?
         a) Raise TypeError (recommended)
         b) Return false
      Answer one per line as <n>: <letter or your words>, e.g. printf '1: a\n' | kogen intent shape is-even -
      Or enter all: recommended to use every question's recommended option.
      """

  Scenario: A question whose options have no recommended mark is refused with feedback instead of failing all: recommended later with answers_invalid
    When k-shape-question-options-recommended-checked: the "no-recommended" case is shaped
    Then k-shape-question-options-recommended-checked: the observations include:
      """
      {"exit": 5,
       "error": "",
       "refused_gates": ["validation"],
       "request_kinds": ["shaper", "shaper", "question_check", "shaper", "decision_check", "literal_check", "contract_check"],
       "questions_json": {"slug": "is-even", "questions": [
         {"number": 1, "question": "How should calc.is_even(n) answer a number that is not an integer?", "kind": "product", "recommended": "a",
          "options": [{"letter": "a", "label": "Raise TypeError", "recommended": true}, {"letter": "b", "label": "Return false", "recommended": false}]}]},
       "state_rounds": [[{"question": "How should calc.is_even(n) answer a number that is not an integer?",
                          "options": [{"label": "Raise TypeError", "recommended": true}, {"label": "Return false"}]}]],
       "answer_exit": 0,
       "answer_error": "",
       "answer_error_lines": [],
       "receipt_answers": {"1": "a"},
       "receipt_resolved_answers": {"1": "Raise TypeError"},
       "published": true}
      """
    And k-shape-question-options-recommended-checked: the observation "feedback_q1" is exactly:
      """
      invalid submission:
      question 1 (How should calc.is_even(n) answer a number that is not an integer?): no option is marked recommended
      every question needs at least two options with distinct, non-empty labels and exactly one option marked recommended
      No proposals from this submission were retained; earlier proposals are unchanged.
      """
    And k-shape-question-options-recommended-checked: the observation "stdout" is exactly these lines:
      """
      Questions for is-even:
      1. How should calc.is_even(n) answer a number that is not an integer?
         a) Raise TypeError (recommended)
         b) Return false
      Answer one per line as <n>: <letter or your words>, e.g. printf '1: a\n' | kogen intent shape is-even -
      Or enter all: recommended to use every question's recommended option.
      """

  Scenario: A question with two options both marked recommended is refused with feedback naming both
    When k-shape-question-options-recommended-checked: the "two-recommended" case is shaped
    Then k-shape-question-options-recommended-checked: the observations include:
      """
      {"exit": 5,
       "error": "",
       "refused_gates": ["validation"],
       "request_kinds": ["shaper", "shaper", "question_check"],
       "questions_json": {"slug": "is-even", "questions": [
         {"number": 1, "question": "How should calc.is_even(n) answer a number that is not an integer?", "kind": "product", "recommended": "a",
          "options": [{"letter": "a", "label": "Raise TypeError", "recommended": true}, {"letter": "b", "label": "Return false", "recommended": false}]}]},
       "state_rounds": [[{"question": "How should calc.is_even(n) answer a number that is not an integer?",
                          "options": [{"label": "Raise TypeError", "recommended": true}, {"label": "Return false"}]}]]}
      """
    And k-shape-question-options-recommended-checked: the observation "feedback_q1" is exactly:
      """
      invalid submission:
      question 1 (How should calc.is_even(n) answer a number that is not an integer?): 2 options are marked recommended (a, b)
      every question needs at least two options with distinct, non-empty labels and exactly one option marked recommended
      No proposals from this submission were retained; earlier proposals are unchanged.
      """
    And k-shape-question-options-recommended-checked: the observation "stdout" is exactly these lines:
      """
      Questions for is-even:
      1. How should calc.is_even(n) answer a number that is not an integer?
         a) Raise TypeError (recommended)
         b) Return false
      Answer one per line as <n>: <letter or your words>, e.g. printf '1: a\n' | kogen intent shape is-even -
      Or enter all: recommended to use every question's recommended option.
      """

  Scenario: A question with a repeated option label is refused with feedback naming the repeat
    When k-shape-question-options-recommended-checked: the "repeated-label" case is shaped
    Then k-shape-question-options-recommended-checked: the observations include:
      """
      {"exit": 5,
       "error": "",
       "refused_gates": ["validation"],
       "request_kinds": ["shaper", "shaper", "question_check"],
       "state_rounds": [[{"question": "How should calc.is_even(n) answer a number that is not an integer?",
                          "options": [{"label": "Raise TypeError", "recommended": true}, {"label": "Return false"}]}]]}
      """
    And k-shape-question-options-recommended-checked: the observation "feedback_q1" is exactly:
      """
      invalid submission:
      question 1 (How should calc.is_even(n) answer a number that is not an integer?): option b repeats the label of option a
      every question needs at least two options with distinct, non-empty labels and exactly one option marked recommended
      No proposals from this submission were retained; earlier proposals are unchanged.
      """
    And k-shape-question-options-recommended-checked: the observation "stdout" is exactly these lines:
      """
      Questions for is-even:
      1. How should calc.is_even(n) answer a number that is not an integer?
         a) Raise TypeError (recommended)
         b) Return false
      Answer one per line as <n>: <letter or your words>, e.g. printf '1: a\n' | kogen intent shape is-even -
      Or enter all: recommended to use every question's recommended option.
      """

  Scenario: A question with an empty option label is refused with feedback naming the empty label
    When k-shape-question-options-recommended-checked: the "empty-label" case is shaped
    Then k-shape-question-options-recommended-checked: the observations include:
      """
      {"exit": 5,
       "error": "",
       "refused_gates": ["validation"],
       "request_kinds": ["shaper", "shaper", "question_check"],
       "state_rounds": [[{"question": "How should calc.is_even(n) answer a number that is not an integer?",
                          "options": [{"label": "Raise TypeError", "recommended": true}, {"label": "Return false"}]}]]}
      """
    And k-shape-question-options-recommended-checked: the observation "feedback_q1" is exactly:
      """
      invalid submission:
      question 1 (How should calc.is_even(n) answer a number that is not an integer?): option b has an empty label
      every question needs at least two options with distinct, non-empty labels and exactly one option marked recommended
      No proposals from this submission were retained; earlier proposals are unchanged.
      """

  Scenario: A question with a single option is refused with feedback, because a meaningful question has at least two options
    When k-shape-question-options-recommended-checked: the "one-option" case is shaped
    Then k-shape-question-options-recommended-checked: the observations include:
      """
      {"exit": 5,
       "error": "",
       "refused_gates": ["validation"],
       "request_kinds": ["shaper", "shaper", "question_check"],
       "state_rounds": [[{"question": "How should calc.is_even(n) answer a number that is not an integer?",
                          "options": [{"label": "Raise TypeError", "recommended": true}, {"label": "Return false"}]}]]}
      """
    And k-shape-question-options-recommended-checked: the observation "feedback_q1" is exactly:
      """
      invalid submission:
      question 1 (How should calc.is_even(n) answer a number that is not an integer?): only one option
      every question needs at least two options with distinct, non-empty labels and exactly one option marked recommended
      No proposals from this submission were retained; earlier proposals are unchanged.
      """

  Scenario: One malformed question refuses its whole submission, the feedback names only that question, and the repaired pair is asked together
    When k-shape-question-options-recommended-checked: the "second-of-two" case is shaped
    Then k-shape-question-options-recommended-checked: the observations include:
      """
      {"exit": 5,
       "error": "",
       "refused_gates": ["validation"],
       "request_kinds": ["shaper", "shaper", "question_check"],
       "questions_json": {"slug": "is-even", "questions": [
         {"number": 1, "question": "How should calc.is_even(n) answer a number that is not an integer?", "kind": "product", "recommended": "a",
          "options": [{"letter": "a", "label": "Raise TypeError", "recommended": true}, {"letter": "b", "label": "Return false", "recommended": false}]},
         {"number": 2, "question": "Should calc.is_even(n) accept numbers given as strings?", "kind": "product", "recommended": "a",
          "options": [{"letter": "a", "label": "No, only numbers", "recommended": true}, {"letter": "b", "label": "Yes, parse them", "recommended": false}]}]}}
      """
    And k-shape-question-options-recommended-checked: the observation "feedback_q1" is exactly:
      """
      invalid submission:
      question 2 (Should calc.is_even(n) accept numbers given as strings?): no option is marked recommended
      every question needs at least two options with distinct, non-empty labels and exactly one option marked recommended
      No proposals from this submission were retained; earlier proposals are unchanged.
      """
    And k-shape-question-options-recommended-checked: the observation "stdout" is exactly these lines:
      """
      Questions for is-even:
      1. How should calc.is_even(n) answer a number that is not an integer?
         a) Raise TypeError (recommended)
         b) Return false
      2. Should calc.is_even(n) accept numbers given as strings?
         a) No, only numbers (recommended)
         b) Yes, parse them
      Answer one per line as <n>: <letter or your words>, e.g. printf '1: a\n2: a\n' | kogen intent shape is-even -
      Or enter all: recommended to use every question's recommended option.
      """

  Scenario: Malformed question submissions stop at the existing limit of 3 attempts, naming the malformed question, with nothing printed or pending
    When k-shape-question-options-recommended-checked: the "bound" case is shaped
    Then k-shape-question-options-recommended-checked: the observations include:
      """
      {"exit": 1,
       "error": "shape/invalid_output: the shaper's output was not a valid Intent after 3 attempts",
       "error_lines": [
         "question 1 (How should calc.is_even(n) answer a number that is not an integer?): 2 options are marked recommended (a, b)",
         "every question needs at least two options with distinct, non-empty labels and exactly one option marked recommended",
         "earliest unresolved refusal (attempt 1, gate validation):",
         "question 1 (How should calc.is_even(n) answer a number that is not an integer?): no options",
         "every question needs at least two options with distinct, non-empty labels and exactly one option marked recommended"],
       "stdout": "",
       "refused_gates": ["validation", "validation", "validation"],
       "request_kinds": ["shaper", "shaper", "shaper"],
       "questions_json": null,
       "state_rounds": null}
      """
    And k-shape-question-options-recommended-checked: the observation "feedback_q2" is exactly:
      """
      invalid submission:
      question 1 (How should calc.is_even(n) answer a number that is not an integer?): no option is marked recommended
      every question needs at least two options with distinct, non-empty labels and exactly one option marked recommended
      No proposals from this submission were retained; earlier proposals are unchanged.
      """

  Scenario: A well-formed handoff question with (a) recommended is asked, and all: recommended selects (a) in text and machine output alike
    When k-shape-question-options-recommended-checked: the "handoff" case is shaped
    Then k-shape-question-options-recommended-checked: the observations include:
      """
      {"exit": 5,
       "error": "",
       "refused_gates": [],
       "request_kinds": ["shaper", "question_check", "shaper", "decision_check", "literal_check", "contract_check"],
       "questions_json": {"slug": "is-even", "questions": [
         {"number": 1, "question": "Sign in to the npm registry in your browser so that acceptance can install the is-even test tools.", "kind": "handoff", "recommended": "a",
          "options": [{"letter": "a", "label": "I have signed in", "recommended": true}, {"letter": "b", "label": "Pause shaping until I have signed in", "recommended": false}]}]},
       "answer_exit": 0,
       "answer_error": "",
       "answer_last_stdout_line": "Next: kogen intent approve is-even",
       "receipt_answers": {"1": "a"},
       "receipt_resolved_answers": {"1": "I have signed in"},
       "pending_after_answer": false,
       "published": true}
      """
    And k-shape-question-options-recommended-checked: the observation "stdout" is exactly these lines:
      """
      Questions for is-even:
      This question is an authentication or security handoff, not a design choice.
      1. Sign in to the npm registry in your browser so that acceptance can install the is-even test tools.
         a) I have signed in (recommended)
         b) Pause shaping until I have signed in
      Answer one per line as <n>: <letter or your words>, e.g. printf '1: a\n' | kogen intent shape is-even -
      Or enter all: recommended to use every question's recommended option.
      """
    And k-shape-question-options-recommended-checked: the observation "earlier_round" is exactly:
      """
      Round 1:
      1. Sign in to the npm registry in your browser so that acceptance can install the is-even test tools.
         a) I have signed in
         b) Pause shaping until I have signed in
         answer: 1: a
      """

  Scenario: A well-formed UI/UX question is asked as today with the same provider requests, and an explicit answer continues shaping
    When k-shape-question-options-recommended-checked: the "ui" case is shaped
    Then k-shape-question-options-recommended-checked: the observations include:
      """
      {"exit": 5,
       "error": "",
       "refused_gates": [],
       "request_kinds": ["shaper", "question_check", "shaper", "decision_check", "literal_check", "contract_check"],
       "questions_json": {"slug": "is-even", "questions": [
         {"number": 1, "question": "How should calc.is_even(n) answer a number that is not an integer?", "kind": "product", "recommended": "a",
          "options": [{"letter": "a", "label": "Raise TypeError", "recommended": true}, {"letter": "b", "label": "Return false", "recommended": false}]}]},
       "answer_exit": 0,
       "answer_error": "",
       "answer_last_stdout_line": "Next: kogen intent approve is-even",
       "receipt_answers": {"1": "b"},
       "receipt_resolved_answers": {"1": "Return false"},
       "published": true}
      """
    And k-shape-question-options-recommended-checked: the observation "stdout" is exactly these lines:
      """
      Questions for is-even:
      1. How should calc.is_even(n) answer a number that is not an integer?
         a) Raise TypeError (recommended)
         b) Return false
      Answer one per line as <n>: <letter or your words>, e.g. printf '1: a\n' | kogen intent shape is-even -
      Or enter all: recommended to use every question's recommended option.
      """
    And k-shape-question-options-recommended-checked: the observation "question_check_input_1" is exactly:
      """
      Slug: is-even

      # Request

      Add calc.is_even(n), which returns true for even integers.


      # Earlier questions and answers

      none

      # Questions

      1. How should calc.is_even(n) answer a number that is not an integer?
         a) Raise TypeError (recommended)
         b) Return false
      """

  Scenario: A well-formed handoff question whose recommended option waits is still asked
    When k-shape-question-options-recommended-checked: the "handoff-wait" case is shaped
    Then k-shape-question-options-recommended-checked: the observations include:
      """
      {"exit": 5,
       "error": "",
       "refused_gates": [],
       "request_kinds": ["shaper", "question_check"],
       "questions_json": {"slug": "is-even", "questions": [
         {"number": 1, "question": "Sign in to the npm registry in your browser so that acceptance can install the is-even test tools.", "kind": "handoff", "recommended": "a",
          "options": [{"letter": "a", "label": "Wait: pause shaping until I have signed in", "recommended": true}, {"letter": "b", "label": "I have signed in", "recommended": false}]}]}}
      """
    And k-shape-question-options-recommended-checked: the observation "stdout" is exactly these lines:
      """
      Questions for is-even:
      This question is an authentication or security handoff, not a design choice.
      1. Sign in to the npm registry in your browser so that acceptance can install the is-even test tools.
         a) Wait: pause shaping until I have signed in (recommended)
         b) I have signed in
      Answer one per line as <n>: <letter or your words>, e.g. printf '1: a\n' | kogen intent shape is-even -
      Or enter all: recommended to use every question's recommended option.
      """
