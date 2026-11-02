Feature: Native shaping keeps calling tools while it finds where an entity's data lives

  The harness features/steps/k_shape_data_stores_no_tool_test.go.txt runs Shape in disposable fixture projects:
  the native shaper against the shape package's scripted local provider, and the external shaper against its
  scripted Claude Code shaper. No real provider, account or benchmark content is involved. The plain fixture is
  the ordinary is-even request. The account fixture keeps an account's memberships in db/memberships.csv, which
  its Account model (lib/account.js) reaches, and one export file per account, exports/1.json and exports/2.json,
  written by lib/exports.js, which the model does not reach; lib/notes.txt holds only my_account_id and
  account_ids, and .kogen/notes.txt is Kogen's own directory. The scripted shaper first submits the package
  scoped to the model's associations, then the package that covers every store.

  Scenario: An ordinary request on the native lane calls tools and publishes a valid Intent with the base shaper instructions
    When the data-stores harness shapes in mode "plain"
    Then the data-stores shaping observations include:
      """
      {"exit": 0, "error": "", "shaper_requests": 1,
       "tools": "read_file, view_image, list_files, search_text, probe, documentation, data_stores, propose_files, submit",
       "data_stores_tool": {"type": "function", "name": "data_stores",
         "description": "Find where an entity's data lives. For a request about an entity's data (for example deleting, exporting or moving an account's data), call this with each spelling of the entity's id key before you write scope and acceptance. It lists every file tracked at HEAD that references a key and how intent.md's ## Data stores section covers each one.",
         "parameters": {"type": "object", "properties": {"keys": {"type": "array", "items": {"type": "string"}}}, "required": ["keys"], "additionalProperties": false},
         "strict": false},
       "instructions_sha256": "48a1bf625e1b71593f687d64b7a6d85c1c30749180462b97917e83084d1014c4",
       "instructions_mention_data_stores": false,
       "refused_gates": [], "no_tool_turns": 0, "acceptance_runs": 2, "published": true,
       "last_stdout_line": "Next: kogen intent approve is-even"}
      """

  Scenario: A native request about an account's data runs the store scan as a tool and the gate refuses the package that leaves stores uncovered
    When the data-stores harness shapes in mode "stores"
    Then the data-stores shaping observations include:
      """
      {"exit": 0, "error": "", "shaper_requests": 3,
       "tools": "read_file, view_image, list_files, search_text, probe, documentation, data_stores, propose_files, submit",
       "instructions_sha256": "48a1bf625e1b71593f687d64b7a6d85c1c30749180462b97917e83084d1014c4",
       "instructions_mention_data_stores": false,
       "refused_gates": ["data_stores"], "no_tool_turns": 0, "acceptance_runs": 2, "published": true,
       "last_stdout_line": "Next: kogen intent approve is-even"}
      """
    And the data-stores shaping observation "data_stores_output" is exactly:
      """
      Data stores for keys account_id: files tracked at HEAD that reference a key
      - db/memberships.csv references account_id (line 1)
      - exports/1.json references account_id (line 1)
      - exports/2.json references account_id (line 1)
      - lib/account.js references account_id (line 1)
      - lib/exports.js references account_id (lines 1, 4)
      Record them in a ## Data stores section of intent.md: a line - key `<key>` for each key, then, for each file above or a directory (a path ending in /) that holds some of them, either - `<path>`: in scope, observed by `<text>`, where acceptance.feature or a step file listed under Acceptance contains <text>, or - `<path>`: excluded by decision <n>, where line <n> of ## Decisions names <path> and keeps its source. Shape refuses a candidate whose section leaves one of these files uncovered.
      """
    And the data-stores shaping observation "first_submission_feedback" is exactly:
      """
      Data stores: 3 problem(s) with where the data of the declared keys lives; nothing was published. Every file tracked at HEAD that references a key declared under ## Data stores needs an item that covers it: in scope, observed by text that acceptance.feature or a listed step file contains, or excluded by a numbered Decision that names it. Resubmitting this candidate unchanged is refused again.
      - exports/1.json references account_id (line 1); no ## Data stores item covers it
      - exports/2.json references account_id (line 1); no ## Data stores item covers it
      - lib/exports.js references account_id (lines 1, 4); no ## Data stores item covers it
      """

  Scenario: The external shaper gets the data-stores guidance in its session text and the same refusal as its next round's feedback
    When the data-stores harness shapes in mode "external"
    Then the data-stores shaping observations include:
      """
      {"exit": 0, "error": "", "shaper_requests": 0, "shaper_sessions": 1, "shaper_rounds": 2,
       "refused_gates": ["data_stores"], "acceptance_runs": 2, "published": true,
       "last_stdout_line": "Next: kogen intent approve is-even"}
      """
    And the data-stores shaping observation "external_data_stores_paragraph" is exactly:
      """
      Data stores: for a request about an entity's data (for example deleting, exporting or moving an account's data), establish where that data lives before you write scope and acceptance: its schemas, the associations its model reaches, and every other store keyed by the entity's id (other tables, files, directories, indexes), found by searching the repository for each spelling of the id key. Record them in a ## Data stores section of intent.md: a line - key `<key>` for each spelling, then, for each file or directory (a path ending in /) that references a key, either - `<path>`: in scope, observed by `<text>`, where acceptance.feature or a step file listed under Acceptance contains <text>, or - `<path>`: excluded by decision <n>, where line <n> of ## Decisions names <path> and keeps its source. Shape refuses a candidate whose section leaves uncovered a file tracked at HEAD that references a declared key.
      """
    And the data-stores shaping observation "second_round_feedback" is exactly:
      """
      Data stores: 3 problem(s) with where the data of the declared keys lives; nothing was published. Every file tracked at HEAD that references a key declared under ## Data stores needs an item that covers it: in scope, observed by text that acceptance.feature or a listed step file contains, or excluded by a numbered Decision that names it. Resubmitting this candidate unchanged is refused again.
      - exports/1.json references account_id (line 1); no ## Data stores item covers it
      - exports/2.json references account_id (line 1); no ## Data stores item covers it
      - lib/exports.js references account_id (lines 1, 4); no ## Data stores item covers it
      """
