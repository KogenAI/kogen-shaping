Feature: Shaping decides every non-UX question itself and asks only open UI/UX questions

  The harness features/steps/k_shape_decide_non_ux_test.go.txt runs Shape in disposable fixture projects:
  the native shaper against the shape package's scripted local provider, and the external shaper against its
  scripted Claude Code shaper. The scripted provider also answers Kogen's question check. The fixtures are
  synthetic stand-ins for the private AoR task kits: a billing project whose Account model already has a
  comped? stub (app/models/account.rb), whose billing selects active and past-due accounts (lib/billing.rb) and
  whose charges go through a gateway module (lib/gateway.rb), and the plain is-even project for the UI/UX case.
  Every scripted question carries a recommended option, and no scripted question is preceded by a probe.

  Scenario: Non-UX questions asked without probing are refused, then decided with Decisions that cite the repository
    When the decide-non-ux harness shapes in mode "decides"
    Then the decide-non-ux observations include:
      """
      {"exit": 0,
       "error": "",
       "shaper_requests": 4,
       "question_check_requests": 1,
       "question_check_classes": [["repository_fact", "product_policy", "environment"]],
       "tools": "read_file, view_image, list_files, search_text, probe, documentation, data_stores, propose_files, submit",
       "instructions_sha256": "48a1bf625e1b71593f687d64b7a6d85c1c30749180462b97917e83084d1014c4",
       "question_check_tools": "submit_question_check",
       "question_check_tool": {"description": "Submit the question check: one item per question with its number, class and reason.", "name": "submit_question_check", "parameters": {"additionalProperties": false, "properties": {"items": {"items": {"additionalProperties": false, "properties": {"class": {"enum": ["ui_ux", "handoff", "answered", "technical", "product_policy", "environment", "repository_fact"], "type": "string"}, "number": {"type": "integer"}, "why": {"type": "string"}}, "required": ["number", "class", "why"], "type": "object"}, "type": "array"}}, "required": ["items"], "type": "object"}, "strict": true, "type": "function"},
       "refused_gates": ["question_check", "decision_evidence"],
       "questions_pending": false,
       "published": true,
       "last_stdout_line": "Next: kogen intent approve is-even"}
      """
    And the decide-non-ux observation "question_check_instructions" is exactly:
      """
      You check the questions a shaper wants to ask the user about one feature request. Only UI/UX choices that the request and the user's answers leave open, and mandatory authentication or security handoffs, may reach the user; the shaper investigates and decides everything else itself. You see only the request text, the earlier questions with the user's answers, and the questions. Read-only: do not use any tool except submit_question_check.

      Give one item for every question: its number, one class and why (one short reason). The classes:
      - ui_ux: a choice of what the user sees or does in the product's interface or interaction (layout, wording, flow, feedback, input behavior) that the request and the earlier answers leave open;
      - handoff: a mandatory authentication or security step that only the user can perform, such as signing in or granting access;
      - answered: the request or an earlier answer already settles it;
      - technical: an implementation, architecture, library, protocol, data or testing choice;
      - product_policy: a business, pricing, permission or other product rule that is not a choice of interface or interaction;
      - environment: whether a tool, service, account, credential, network or other resource is available where the work runs;
      - repository_fact: a fact that the repository's files, history or commands can establish.
      Answer by calling submit_question_check.
      """
    And the decide-non-ux observation "question_check_input" is exactly:
      """
      Slug: is-even

      # Request

      Monthly billing must not charge comped accounts: `bin/bill` charges every other billable account its plan price.


      # Earlier questions and answers

      none

      # Questions

      1. Is there already a way to mark an account as comped, or should billing add a comped flag?
         a) Add a comped column to accounts (recommended)
         b) Reuse an existing marker
      2. Should billing charge past-due accounts?
         a) Yes, like active accounts (recommended)
         b) No, skip them
      3. Is the Stripe CLI installed where billing runs?
         a) Assume it is installed (recommended)
         b) Assume it is not
      """
    And the decide-non-ux observation "feedback_q1" is exactly:
      """
      Question check: 3 of 3 question(s) are not UI/UX choices left to the user; nothing was asked. Investigate and decide each one yourself: read the repository or run a probe, then record a Decisions line that cites what you read or ran before its source tag, as - <decision> (evidence: <repository path>) (source: request) or - <decision> (evidence: ran `<command>`) (source: request). Ask the user only UI/UX questions that the request and the answers leave open, or a mandatory authentication or security handoff.
      - question 1 (repository_fact): Is there already a way to mark an account as comped, or should billing add a comped flag? | why: synthetic repository_fact question | decide it from what the repository and the request establish
      - question 2 (product_policy): Should billing charge past-due accounts? | why: synthetic product_policy question | decide it from what the repository and the request establish
      - question 3 (environment): Is the Stripe CLI installed where billing runs? | why: synthetic environment question | do not ask about the environment; proceed without it and record that Decision, or, if the request cannot be shaped without it, submit only such environment questions again and Shape ends with an environment fault
      """
    And the decide-non-ux observation "feedback_f1" is exactly:
      """
      Decision evidence: 2 problem(s) with what the Decisions cite; nothing was published. A Decision's (evidence: ...) names files or directories tracked at HEAD, or ran `<command>` for a command this shaping ran, separated by "; ". Resubmitting this candidate unchanged is refused again.
      - decision 1: app/models/comp.rb is not a file or directory tracked at HEAD
      - decision 3: no probe of this shaping ran `stripe --version`
      """

  Scenario: A genuine UI/UX question the user has not answered is still asked, and its answer completes shaping
    When the decide-non-ux harness shapes in mode "asks-ui"
    Then the decide-non-ux observations include:
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
    And the decide-non-ux observation "first_stdout" is exactly these lines:
      """
      Questions for is-even:
      1. When a reaction is longer than 16 characters, should the field stop accepting characters at 16, or accept them and show an error on submit?
         a) Stop accepting characters at 16 (recommended)
         b) Accept them and show an error on submit
      Answer one per line as <n>: <letter or your words>, e.g. printf '1: a\n' | kogen intent shape is-even -
      Or enter all: recommended to use every question's recommended option.
      """
    And the decide-non-ux observation "question_check_input" is exactly:
      """
      Slug: is-even

      # Request

      Let users react to a comment with a short text reaction of at most 16 characters.


      # Earlier questions and answers

      none

      # Questions

      1. When a reaction is longer than 16 characters, should the field stop accepting characters at 16, or accept them and show an error on submit?
         a) Stop accepting characters at 16 (recommended)
         b) Accept them and show an error on submit
      """

  Scenario: An environment question is never asked: asked again alone, it ends Shape with an environment fault
    When the decide-non-ux harness shapes in mode "environment"
    Then the decide-non-ux observations include:
      """
      {"exit": 3,
       "error": "environment/shape_environment: shaping cannot go on without something only your environment can provide; nothing was asked or published",
       "error_lines": ["- Is a Stripe test API key available in this environment?"],
       "shaper_requests": 2,
       "question_check_requests": 2,
       "question_check_classes": [["environment"], ["environment"]],
       "refused_gates": ["question_check", "question_check"],
       "questions_pending": false,
       "intent_exists": false,
       "published": false}
      """
    And the decide-non-ux observation "feedback_q1" is exactly:
      """
      Question check: 1 of 1 question(s) are not UI/UX choices left to the user; nothing was asked. Investigate and decide each one yourself: read the repository or run a probe, then record a Decisions line that cites what you read or ran before its source tag, as - <decision> (evidence: <repository path>) (source: request) or - <decision> (evidence: ran `<command>`) (source: request). Ask the user only UI/UX questions that the request and the answers leave open, or a mandatory authentication or security handoff.
      - question 1 (environment): Is a Stripe test API key available in this environment? | why: synthetic environment question | do not ask about the environment; proceed without it and record that Decision, or, if the request cannot be shaped without it, submit only such environment questions again and Shape ends with an environment fault
      """

  Scenario: The external shaper's non-UX question is refused by an external question check and decided
    When the decide-non-ux harness shapes in mode "external"
    Then the decide-non-ux observations include:
      """
      {"exit": 0,
       "error": "",
       "shaper_requests": 0,
       "shaper_sessions": 1,
       "question_check_sessions": 1,
       "rounds": 3,
       "question_check_classes": [["repository_fact"]],
       "question_check_stable_first_line": "You check the questions a shaper wants to ask the user about one feature request. Only UI/UX choices that the request and the user's answers leave open, and mandatory authentication or security handoffs, may reach the user; the shaper investigates and decides everything else itself. You see only the request text, the earlier questions with the user's answers, and the questions. Read-only: do not use any tool except submit_question_check.",
       "refused_gates": ["question_check"],
       "questions_pending": false,
       "published": true,
       "last_stdout_line": "Next: kogen intent approve is-even"}
      """
    And the decide-non-ux observation "round_3_feedback" is exactly:
      """
      Question check: 1 of 1 question(s) are not UI/UX choices left to the user; nothing was asked. Investigate and decide each one yourself: read the repository or run a probe, then record a Decisions line that cites what you read or ran before its source tag, as - <decision> (evidence: <repository path>) (source: request) or - <decision> (evidence: ran `<command>`) (source: request). Ask the user only UI/UX questions that the request and the answers leave open, or a mandatory authentication or security handoff.
      - question 1 (repository_fact): Is there already a way to mark an account as comped, or should billing add a comped flag? | why: synthetic repository_fact question | decide it from what the repository and the request establish
      """
