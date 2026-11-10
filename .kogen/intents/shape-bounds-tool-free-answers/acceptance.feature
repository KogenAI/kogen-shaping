Feature: Bound native shaping conversations
  This feature reproduces the prose and preparation loop without live providers.
  The historical 10 November 2026 reports involved 1,110 requests with 550 prose
  answers and 549 forced turns, and another conversation of 878 requests.
  Provider timeout and login retry history does not change this local contract.

  Scenario: alternating
    Prose one, read README.md, prose two, read README.md, prose three must
    refuse with exit 1, invalid_output and the existing no-tool-limit text.
    Expect 5 requests, 3 tool_free_answer events, 2 forced_submit_turn events,
    last tool-free answer three, no fourth prose request and no publication.
    Then shape-bounds-tool-free-answers: alternating obeys the bound

  Scenario: reset
    Two prose answers, an invalid mixed questions/files submission, then two
    more prose answers must allow the following valid submission to publish.
    Expect exit 0 and 7 requests including initial discovery.
    Then shape-bounds-tool-free-answers: reset obeys the bound

  Scenario: cap
    Read-only answers must stop at 300 provider requests with exit 1 and no
    publication. Repeat with one earlier prose answer last known answer:
    the cap message must retain it. Expect one shaping_turn_counts event
    with request_cap 300 and requests 300; never send request 301.
    Then shape-bounds-tool-free-answers: cap obeys the bound

  Scenario: consecutive
    Three consecutive prose answers must stop with exit 1, invalid_output,
    the existing no-tool-limit refusal and exactly 3 provider requests.
    Then shape-bounds-tool-free-answers: consecutive obeys the bound
