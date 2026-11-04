Feature: An environment stop needs cited evidence, and spent retries are reported as spent retries

  The harness features/steps/k-shape-environment-stop-evidence__harness.js runs Shape (kogen intent shape) with the native
  shaper against the shape package's scripted local provider, which plays both the shaper and Kogen's question check, in the
  shape tests' is-even project with an invented frost forecast fixture committed on top. The stand-in fixture has
  lib/weather.js (production reads the API key from SKYLINE_API_KEY) and test/support/fake_weather.js (a fake client that
  acceptance uses, no key needed); the live fixture has only lib/weather.js, which says there is no fake client and every call
  reaches the live Skyline service. The scripted question check gives the class and why written in each scenario; the
  scenarios observe what Kogen passes to it and what Kogen does with its answer. A published run publishes the shape tests'
  is-even package with the scoped step file features/steps/is-even__steps.js and its 'is-even: ' step text. The default
  limit of 3 submissions applies.

  Scenario: An environment question about a key that only production needs is refused with "record a Decision and continue", and the next submission's assumption Decision is published
    When k-shape-environment-stop-evidence: the "stand-in" case is shaped
    Then k-shape-environment-stop-evidence: the observations include:
      """
      {"exit": 0,
       "error": "",
       "shaper_requests": 2,
       "question_check_requests": 1,
       "question_check_classes": [["environment"]],
       "refused_gates": ["question_check"],
       "questions_pending": false,
       "published": true,
       "last_stdout_line": "Next: kogen intent approve is-even"}
      """
    And k-shape-environment-stop-evidence: the observation "feedback_q1" is exactly:
      """
      Question check: 1 of 1 question(s) are not UI/UX choices left to the user; nothing was asked. Investigate and decide each one yourself: read the repository or run a probe, then record a Decisions line that cites what you read or ran before its source tag, as - <decision> (evidence: <repository path>) (source: request) or - <decision> (evidence: ran `<command>`) (source: request). Where the repository has no evidence or conflicting evidence, record the uncertainty you decided under instead, as - <decision> (assumption: <the uncertainty>) (source: request); the user confirms or reverses it at approval. Ask the user only UI/UX questions that the request and the answers leave open, or a mandatory authentication or security handoff.
      - question 1 (environment): Is SKYLINE_API_KEY set where bin/frost-forecast runs? | why: the key is needed only to run bin/frost-forecast in production | record a Decision and continue: proceed without it, with the stand-in the repository shows or one the Intent specifies, as - <decision> (assumption: <the uncertainty>) (source: request); Shape stops only for environment questions that cite, as (evidence: <repository path>) or (evidence: ran `<command>`) in the question, what this shaping read or ran that shows the Intent or its acceptance cannot be written or run without the item
      """
    And k-shape-environment-stop-evidence: the observation "question_check_input_1" is exactly:
      """
      Slug: is-even

      # Request

      bin/frost-forecast must print tomorrow's frost risk for the orchard from the Skyline weather API; production runs read the API key from SKYLINE_API_KEY.


      # Earlier questions and answers

      none

      # Questions

      1. Is SKYLINE_API_KEY set where bin/frost-forecast runs?
         a) Assume it is set (recommended)
         b) Assume it is not
      """

  Scenario: A question citing a file that shows a stand-in is not a stop, even twice and even when it calls the service essential, and shaping goes on to a recorded Decision
    When k-shape-environment-stop-evidence: the "contradicted" case is shaped
    Then k-shape-environment-stop-evidence: the observations include:
      """
      {"exit": 0,
       "error": "",
       "shaper_requests": 4,
       "question_check_requests": 2,
       "question_check_classes": [["environment"], ["environment"]],
       "refused_gates": ["question_check", "question_check"],
       "questions_pending": false,
       "published": true,
       "last_stdout_line": "Next: kogen intent approve is-even"}
      """
    And k-shape-environment-stop-evidence: the observation "question_check_input_1" is exactly:
      """
      Slug: is-even

      # Request

      bin/frost-forecast must print tomorrow's frost risk for the orchard from the Skyline weather API; production runs read the API key from SKYLINE_API_KEY.


      # Earlier questions and answers

      none

      # Questions

      1. Is SKYLINE_API_KEY set where bin/frost-forecast runs? A live Skyline service is essential (evidence: test/support/fake_weather.js)
         a) Assume it is set (recommended)
         b) Assume it is not

      # Cited evidence

      Question 1 cites test/support/fake_weather.js (read by this shaping):
      1	// A fake Skyline weather client: acceptance uses it, and no API key is needed.
      2	exports.client = function () {
      3	  return {forecast: () => ({frost: true})}
      4	}
      """
    And k-shape-environment-stop-evidence: the observation "feedback_q2" is exactly:
      """
      Question check: 1 of 1 question(s) are not UI/UX choices left to the user; nothing was asked. Investigate and decide each one yourself: read the repository or run a probe, then record a Decisions line that cites what you read or ran before its source tag, as - <decision> (evidence: <repository path>) (source: request) or - <decision> (evidence: ran `<command>`) (source: request). Where the repository has no evidence or conflicting evidence, record the uncertainty you decided under instead, as - <decision> (assumption: <the uncertainty>) (source: request); the user confirms or reverses it at approval. Ask the user only UI/UX questions that the request and the answers leave open, or a mandatory authentication or security handoff.
      - question 1 (environment): Is SKYLINE_API_KEY set where bin/frost-forecast runs? A live Skyline service is essential (evidence: test/support/fake_weather.js) | why: test/support/fake_weather.js is a fake Skyline client: acceptance uses it, and no API key is needed | record a Decision and continue: proceed without it, with the stand-in the repository shows or one the Intent specifies, as - <decision> (assumption: <the uncertainty>) (source: request); Shape stops only for environment questions that cite, as (evidence: <repository path>) or (evidence: ran `<command>`) in the question, what this shaping read or ran that shows the Intent or its acceptance cannot be written or run without the item
      """

  Scenario: An acceptance that must call the live service with no stand-in stops on the second evidenced environment-only submission, naming the blocked part and the evidence
    When k-shape-environment-stop-evidence: the "live" case is shaped
    Then k-shape-environment-stop-evidence: the observations include:
      """
      {"exit": 3,
       "error": "environment/shape_environment: shaping cannot go on without something only your environment can provide; nothing was asked or published",
       "error_lines": ["- Is SKYLINE_API_KEY set where the acceptance of bin/frost-forecast runs? Its acceptance must call the live Skyline service (evidence: lib/weather.js; ran `ls test/support 2>/dev/null || echo no test/support folder`)", "  blocks: the acceptance step that calls the live Skyline service cannot run without SKYLINE_API_KEY: lib/weather.js has no fake client, and the probe found no test/support folder", "  evidence: lib/weather.js; ran `ls test/support 2>/dev/null || echo no test/support folder`"],
       "shaper_requests": 3,
       "question_check_requests": 2,
       "question_check_classes": [["environment_prerequisite"], ["environment_prerequisite"]],
       "refused_gates": ["question_check", "question_check"],
       "questions_pending": false,
       "intent_exists": false,
       "published": false,
       "question_check_class_enum": ["ui_ux", "handoff", "answered", "technical", "product_policy", "environment", "environment_prerequisite", "repository_fact"]}
      """
    And k-shape-environment-stop-evidence: the observation "question_check_input_1" is exactly:
      """
      Slug: is-even

      # Request

      bin/frost-forecast must print tomorrow's frost risk for the orchard from the Skyline weather API, and its acceptance must call the live Skyline service with the API key from SKYLINE_API_KEY.


      # Earlier questions and answers

      none

      # Questions

      1. Is SKYLINE_API_KEY set where the acceptance of bin/frost-forecast runs? Its acceptance must call the live Skyline service (evidence: lib/weather.js; ran `ls test/support 2>/dev/null || echo no test/support folder`)
         a) Assume it is set (recommended)
         b) Assume it is not

      # Cited evidence

      Question 1 cites lib/weather.js (read by this shaping):
      1	// The Skyline weather client. There is no fake client: every call reaches the live Skyline service with the API key from SKYLINE_API_KEY.
      2	exports.client = function () {
      3	  return {key: process.env.SKYLINE_API_KEY}
      4	}

      Question 1 cites ran `ls test/support 2>/dev/null || echo no test/support folder` (probe output):
      exit status: 0
      timed_out: false
      wall: <wall> s
      no test/support folder
      """
    And k-shape-environment-stop-evidence: the observation "feedback_q1" is exactly:
      """
      Question check: 1 of 1 question(s) are not UI/UX choices left to the user; nothing was asked. Investigate and decide each one yourself: read the repository or run a probe, then record a Decisions line that cites what you read or ran before its source tag, as - <decision> (evidence: <repository path>) (source: request) or - <decision> (evidence: ran `<command>`) (source: request). Where the repository has no evidence or conflicting evidence, record the uncertainty you decided under instead, as - <decision> (assumption: <the uncertainty>) (source: request); the user confirms or reverses it at approval. Ask the user only UI/UX questions that the request and the answers leave open, or a mandatory authentication or security handoff.
      - question 1 (environment_prerequisite): Is SKYLINE_API_KEY set where the acceptance of bin/frost-forecast runs? Its acceptance must call the live Skyline service (evidence: lib/weather.js; ran `ls test/support 2>/dev/null || echo no test/support folder`) | why: the acceptance step that calls the live Skyline service cannot run without SKYLINE_API_KEY: lib/weather.js has no fake client, and the probe found no test/support folder | the cited evidence shows the Intent or its acceptance cannot be written or run without it; if no stand-in can be used or specified, submit only such evidenced environment questions again and Shape stops with an environment fault that names the blocked part and the evidence; otherwise record a Decision and continue
      """
    And k-shape-environment-stop-evidence: the observation "question_check_instructions" is exactly:
      """
      You check the questions a shaper wants to ask the user about one feature request. Only UI/UX choices that the request and the user's answers leave open, and mandatory authentication or security handoffs, may reach the user; the shaper investigates and decides everything else itself. You see only the request text, the earlier questions with the user's answers, the questions, and the cited evidence: the text this shaping read or ran for each (evidence: ...) item a question cites. Read-only: do not use any tool except submit_question_check.

      Give one item for every question: its number, one class and why (one short reason). The classes:
      - ui_ux: a choice of what the user sees or does in the product's interface or interaction (layout, wording, flow, feedback, input behavior) that the request and the earlier answers leave open;
      - handoff: a mandatory authentication or security step that only the user can perform, such as signing in or granting access;
      - answered: the request or an earlier answer already settles it;
      - technical: an implementation, architecture, library, protocol, data or testing choice;
      - product_policy: a business, pricing, permission or other product rule that is not a choice of interface or interaction;
      - environment: whether a tool, service, account, credential, network or other resource is available where the work runs, when it is not an environment_prerequisite;
      - environment_prerequisite: a resource without which this shaping cannot write the Intent or run its acceptance, as the question's cited evidence shows: the cited text is about the resource the question names and shows it needed and missing, and neither the cited text, the request nor the earlier answers show it optional, needed only to run the finished feature against a real external service, or already stood in for by a fake, stub, simulation or recorded response; the why names the part of the Intent or acceptance that cannot be written or run without it and what in the cited text shows that. A question with no cited text shown, or whose cited text does not support the claimed necessity, is environment, whatever its own wording claims;
      - repository_fact: a fact that the repository's files, history or commands can establish.
      Answer by calling submit_question_check.
      """

  Scenario: Two product_policy refusals and then one environment-only submission on the last attempt end as spent retries with their classes, not as an environment fault
    When k-shape-environment-stop-evidence: the "spent" case is shaped
    Then k-shape-environment-stop-evidence: the observations include:
      """
      {"exit": 1,
       "error": "shape/invalid_output: the shaper's output was not a valid Intent after 3 attempts",
       "error_last_line": "question check refusals: product_policy x2, environment x1",
       "shaper_requests": 3,
       "question_check_requests": 3,
       "question_check_classes": [["product_policy"], ["product_policy"], ["environment"]],
       "refused_gates": ["question_check", "question_check", "question_check"],
       "questions_pending": false,
       "intent_exists": false,
       "published": false}
      """

  Scenario: An evidenced environment-only submission on the last attempt after two product_policy refusals also ends as spent retries
    When k-shape-environment-stop-evidence: the "spent-evidenced" case is shaped
    Then k-shape-environment-stop-evidence: the observations include:
      """
      {"exit": 1,
       "error": "shape/invalid_output: the shaper's output was not a valid Intent after 3 attempts",
       "error_last_line": "question check refusals: product_policy x2, environment_prerequisite x1",
       "shaper_requests": 4,
       "question_check_requests": 3,
       "question_check_classes": [["product_policy"], ["product_policy"], ["environment_prerequisite"]],
       "refused_gates": ["question_check", "question_check", "question_check"],
       "questions_pending": false,
       "intent_exists": false,
       "published": false}
      """

  Scenario: A UI/UX question after the same two refusals is still asked
    When k-shape-environment-stop-evidence: the "spent-ui" case is shaped
    Then k-shape-environment-stop-evidence: the observations include:
      """
      {"exit": 5,
       "error": "",
       "shaper_requests": 3,
       "question_check_requests": 3,
       "question_check_classes": [["product_policy"], ["product_policy"], ["ui_ux"]],
       "refused_gates": ["question_check", "question_check"],
       "questions_pending": true,
       "intent_exists": false}
      """
    And k-shape-environment-stop-evidence: the observation "stdout" is exactly these lines:
      """
      Questions for is-even:
      1. Should bin/frost-forecast print the risk as a percentage or as a word (low, medium, high)?
         a) A percentage (recommended)
         b) A word
      Answer one per line as <n>: <letter or your words>, e.g. printf '1: a\n' | kogen intent shape is-even -
      Or enter all: recommended to use every question's recommended option.
      """

  Scenario: The recorded fixtures of k-shape-decide-non-ux-general are unchanged
    Then k-shape-environment-stop-evidence: these files have exactly these SHA-256 digests:
      """
      d2850fad9b78d2792a2a44164afa10f4adbf57d4d02bde82f2209911c1237394  .kogen/intents/k-shape-decide-non-ux-general/intent.md
      2340717d17d3460bb92b83c0c1d5249c960593bb55563161d17510177739ad23  .kogen/intents/k-shape-decide-non-ux-general/acceptance.feature
      57e2427bf019646f9f5172912df435bdbe091483e433d1bee3e7c997579b3843  features/steps/k_shape_decide_general_steps.js
      674608edab18eca9872669e28277b0df6c644bec64735b9721720ab2bf309653  features/steps/k_shape_decide_general_test.go.txt
      eb5a552a3fd68c66e0c600afe1312e90b3bb1ec2e26e231c495526ce0676fdea  features/steps/testdata/k_shape_decide_general/recorded.json
      """
