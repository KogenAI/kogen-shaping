Feature: Shaping decides non-UX questions on cited evidence it read or ran, and records the uncertainty it decided under

  The harness features/steps/k_shape_decide_general_test.go.txt runs Shape in disposable fixture projects:
  the native shaper against the shape package's scripted local provider, and the external shaper against its
  scripted Claude Code shaper. The scripted provider also answers Kogen's question check. The fixture is an
  invented orchard frost alarm committed onto the shape tests' is-even project: lib/sensors.js stores readings
  in tenths of a degree Celsius, lib/alarm.js compares the latest reading with a limit, and lib/siren.js
  simulates the siren without a hardware driver. The UI/UX case uses the plain is-even project. Every scripted
  question carries a recommended option, and no scripted question is preceded by a read or a probe.

  Scenario: Non-UX questions are refused, then decided on evidence the shaper read or ran, with the uncertain one marked and shown for approval
    When the neutral decide harness shapes in mode "decides"
    Then the neutral decide observations include:
      """
      {"exit": 0,
       "error": "",
       "shaper_requests": 4,
       "question_check_requests": 1,
       "question_check_classes": [["repository_fact", "product_policy", "environment"]],
       "tools": "read_file, view_image, list_files, search_text, probe, documentation, propose_files, submit",
       "instructions_sha256": "48a1bf625e1b71593f687d64b7a6d85c1c30749180462b97917e83084d1014c4",
       "question_check_tools": "submit_question_check",
       "refused_gates": ["question_check", "decision_evidence"],
       "questions_pending": false,
       "published": true,
       "last_stdout_line": "Next: kogen intent approve is-even",
       "card_exit": 5,
       "card_error": "",
       "card_assumptions": ["A block with no reading in the last hour is skipped (assumption: neither the request nor the repository says whether a block without readings is warned about)"]}
      """
    And the neutral decide observation "question_check_input" is exactly:
      """
      Slug: is-even

      # Request

      `bin/frost` must warn about every orchard block whose average reading over the last hour is below 2 °C.


      # Earlier questions and answers

      none

      # Questions

      1. In which unit are the orchard sensor readings stored?
         a) Degrees Celsius (recommended)
         b) Tenths of a degree Celsius
      2. Should a block with no reading in the last hour be warned about?
         a) Yes, warn about it (recommended)
         b) No, skip it
      3. Is a siren hardware driver installed where bin/frost runs?
         a) Assume it is installed (recommended)
         b) Assume it is not
      """
    And the neutral decide observation "feedback_q1" is exactly:
      """
      Question check: 3 of 3 question(s) are not UI/UX choices left to the user; nothing was asked. Investigate and decide each one yourself: read the repository or run a probe, then record a Decisions line that cites what you read or ran before its source tag, as - <decision> (evidence: <repository path>) (source: request) or - <decision> (evidence: ran `<command>`) (source: request). Where the repository has no evidence or conflicting evidence, record the uncertainty you decided under instead, as - <decision> (assumption: <the uncertainty>) (source: request); the user confirms or reverses it at approval. Ask the user only UI/UX questions that the request and the answers leave open, or a mandatory authentication or security handoff.
      - question 1 (repository_fact): In which unit are the orchard sensor readings stored? | why: synthetic repository_fact question | decide it from what the repository and the request establish
      - question 2 (product_policy): Should a block with no reading in the last hour be warned about? | why: synthetic product_policy question | decide it from what the repository and the request establish
      - question 3 (environment): Is a siren hardware driver installed where bin/frost runs? | why: synthetic environment question | do not ask about the environment; proceed without it and record that Decision, or, if the request cannot be shaped without it, submit only such environment questions again and Shape ends with an environment fault
      """
    And the neutral decide observation "feedback_f1" is exactly:
      """
      Decision evidence: 3 problem(s) with what the Decisions cite; nothing was published. A Decision's (evidence: ...) names files this shaping read or directories it listed, tracked at HEAD, or ran `<command>` for a command this shaping ran, separated by "; ". Where the repository has no evidence or conflicting evidence, mark the Decision (assumption: <the uncertainty you decided under>) instead. Resubmitting this candidate unchanged is refused again.
      - decision 1: this shaping did not read lib/alarm.js
      - decision 2: lib/blocks.js is not a file or directory tracked at HEAD
      - decision 3: no probe of this shaping ran `ls -l lib`
      """

  Scenario: A genuine UI/UX question the user has not answered is still asked, and its answer completes shaping
    When the neutral decide harness shapes in mode "asks-ui"
    Then the neutral decide observations include:
      """
      {"first_exit": 5,
       "first_error": "",
       "first_questions_pending": true,
       "exit": 0,
       "error": "",
       "shaper_requests": 2,
       "question_check_requests": 1,
       "question_check_classes": [["ui_ux"]],
       "refused_gates": [],
       "questions_pending": false,
       "published": true,
       "last_stdout_line": "Next: kogen intent approve is-even"}
      """
    And the neutral decide observation "first_stdout" is exactly these lines:
      """
      Questions for is-even:
      1. Should the frost warning appear as a banner across the top of the dashboard or as a badge on the affected block?
         a) A banner across the top (recommended)
         b) A badge on the block
      Answer one per line as <n>: <letter or your words>, e.g. printf '1: a\n' | kogen intent shape is-even -
      Or enter all: recommended to use every question's recommended option.
      """
    And the neutral decide observation "question_check_input" is exactly:
      """
      Slug: is-even

      # Request

      Show a frost warning on the orchard dashboard when a block falls below 2 °C.


      # Earlier questions and answers

      none

      # Questions

      1. Should the frost warning appear as a banner across the top of the dashboard or as a badge on the affected block?
         a) A banner across the top (recommended)
         b) A badge on the block
      """

  Scenario: An environment question is never asked: asked again alone, it ends Shape with an environment fault
    When the neutral decide harness shapes in mode "environment"
    Then the neutral decide observations include:
      """
      {"exit": 3,
       "error": "environment/shape_environment: shaping cannot go on without something only your environment can provide; nothing was asked or published",
       "error_lines": ["- Is a weather service API key available in this environment?"],
       "shaper_requests": 2,
       "question_check_requests": 2,
       "question_check_classes": [["environment"], ["environment"]],
       "refused_gates": ["question_check", "question_check"],
       "questions_pending": false,
       "intent_exists": false,
       "published": false}
      """
    And the neutral decide observation "feedback_q1" is exactly:
      """
      Question check: 1 of 1 question(s) are not UI/UX choices left to the user; nothing was asked. Investigate and decide each one yourself: read the repository or run a probe, then record a Decisions line that cites what you read or ran before its source tag, as - <decision> (evidence: <repository path>) (source: request) or - <decision> (evidence: ran `<command>`) (source: request). Where the repository has no evidence or conflicting evidence, record the uncertainty you decided under instead, as - <decision> (assumption: <the uncertainty>) (source: request); the user confirms or reverses it at approval. Ask the user only UI/UX questions that the request and the answers leave open, or a mandatory authentication or security handoff.
      - question 1 (environment): Is a weather service API key available in this environment? | why: synthetic environment question | do not ask about the environment; proceed without it and record that Decision, or, if the request cannot be shaped without it, submit only such environment questions again and Shape ends with an environment fault
      """

  Scenario: The external shaper's non-UX question is refused by an external question check and decided on what its session read
    When the neutral decide harness shapes in mode "external"
    Then the neutral decide observations include:
      """
      {"exit": 0,
       "error": "",
       "shaper_requests": 0,
       "shaper_sessions": 1,
       "question_check_sessions": 1,
       "rounds": 3,
       "question_check_classes": [["repository_fact"]],
       "refused_gates": ["question_check"],
       "questions_pending": false,
       "published": true,
       "last_stdout_line": "Next: kogen intent approve is-even"}
      """
    And the neutral decide observation "round_3_feedback" is exactly:
      """
      Question check: 1 of 1 question(s) are not UI/UX choices left to the user; nothing was asked. Investigate and decide each one yourself: read the repository or run a probe, then record a Decisions line that cites what you read or ran before its source tag, as - <decision> (evidence: <repository path>) (source: request) or - <decision> (evidence: ran `<command>`) (source: request). Where the repository has no evidence or conflicting evidence, record the uncertainty you decided under instead, as - <decision> (assumption: <the uncertainty>) (source: request); the user confirms or reverses it at approval. Ask the user only UI/UX questions that the request and the answers leave open, or a mandatory authentication or security handoff.
      - question 1 (repository_fact): In which unit are the orchard sensor readings stored? | why: synthetic repository_fact question | decide it from what the repository and the request establish
      """

  Scenario: The landed acceptance with its benchmark-like fixtures is replaced by this neutral one
    Then these files of the landed k-shape-decide-non-ux acceptance do not exist:
      """
      .kogen/intents/k-shape-decide-non-ux/intent.md
      .kogen/intents/k-shape-decide-non-ux/acceptance.feature
      features/steps/k_shape_decide_non_ux_steps.js
      features/steps/k_shape_decide_non_ux_test.go.txt
      """
    And no repository file outside this Intent's acceptance mentions any of these words:
      """
      comped
      billing
      stripe
      reaction
      """
