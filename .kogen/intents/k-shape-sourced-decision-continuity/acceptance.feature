Feature: Sourced decisions are recorded with their evidence and carried across Shape's continuations

  The harness features/steps/k-shape-sourced-decision-continuity__harness.js runs Shape (kogen intent shape) one or more
  times per journey, each later run continuing through the existing answer input, in the shape tests' is-even project with
  an invented parcel audit fixture committed on top: docs/policy.md ("An unreadable parcel record produces an incomplete
  audit observation with safe diagnostics; record bytes are never included."), src/audit.go (the incomplete-observation
  convention audit.Incomplete) and docs/retention.md, which no journey reads. The request is "Make parcel inspection obey
  the existing safety policy; preserve normal reads. The new audit panel's visual presentation is open." The native
  journeys play the shaper and Kogen's question check with the shape package's scripted local provider; the external
  journey plays them with the shape tests' scripted claude-code executor. The scripted question check gives the class,
  why and decision written in each journey; the scenarios observe what Kogen passes to it, what it records and what it
  does with the answers. No live account, credential, external service or paid model is used. Every published journey
  publishes the shape tests' is-even package, whose Decisions carry the recorded decisions' lines.

  Scenario: A security choice worded as a user experience is refused, and its decision is recorded only on evidence this shaping actually read or ran
    When k-shape-sourced-decision-continuity: the "settle" journey is shaped
    Then k-shape-sourced-decision-continuity: the observation "feedback_s1" is exactly:
      """
      Question check: 1 of 1 question(s) are not UI/UX choices left to the user; nothing was asked. Investigate and decide each one yourself: read the repository or run a probe, then record a Decisions line that cites what you read or ran before its source tag, as - <decision> (evidence: <repository path>) (source: request) or - <decision> (evidence: ran `<command>`) (source: request). Where the repository has no evidence or conflicting evidence, record the uncertainty you decided under instead, as - <decision> (assumption: <the uncertainty>) (source: request); the user confirms or reverses it at approval. Ask the user only UI/UX questions that the request and the answers leave open, or a mandatory authentication or security handoff.
      - question 1 (technical): What should a user experience on an unreadable record: reject the operation or save an incomplete observation? | why: rejecting or saving an unreadable record is a security and data-handling choice, whatever its wording | decide it from what the repository and the request establish
      """
    And k-shape-sourced-decision-continuity: the observation "feedback_s2" is exactly:
      """
      Sourced decisions: 3 problem(s) with the decisions this submission records; nothing was recorded, asked or published. A sourced decision cites, as evidence items, files this shaping read (or that an earlier round read and that are unchanged since) or ran `<command>` for a probe of this shaping that succeeded; where there is no such evidence, state the uncertainty you decided under instead. A revision names the decision it revises in revises, says why it changed and cites evidence this round read or ran.
      - sourced decision 1: this shaping did not read docs/retention.md
      - sourced decision 1: the probe of `grep -n 'never included' src/audit.go` failed (exit status: 1), so it is no evidence
      - sourced decision 2: no evidence and no uncertainty; cite what this shaping read or ran, or state the uncertainty you decided under
      """
    And k-shape-sourced-decision-continuity: the observation "ledger_1" is this JSON:
      """
      {"version": 1,
       "decisions": [
        {"id":"D1","round":1,"decision":"An unreadable parcel record produces an incomplete audit observation with safe diagnostics, built with audit.Incomplete","constraints":["record bytes are never included"],"uncertainty":"","revises":"","why":"","superseded_by":"","evidence":[{"item":"docs/policy.md","kind":"read","round":1,"sha256":"6f4f4c3e299d5b91551b2483f35694775247fafc849bfb711c8a631961b70f9a","text":"1\t# Parcel safety policy\n2\t\n3\tAn unreadable parcel record produces an incomplete audit observation with safe diagnostics; record bytes are never included."},{"item":"src/audit.go","kind":"read","round":1,"sha256":"85896eb548276f1995490420465ce5a15b659f2c55b9658c3ea09cda018bbfa7","text":"1\tpackage audit\n2\t\n3\t// Observation is one audit observation of a parcel inspection.\n4\ttype Observation struct {\n5\t\tParcel      string\n6\t\tComplete    bool\n7\t\tDiagnostics []string\n8\t}\n9\t\n10\t// Incomplete is the observation of a parcel whose record could not be read: safe diagnostics only.\n11\tfunc Incomplete(parcel, reason string) Observation {\n12\t\treturn Observation{Parcel: parcel, Diagnostics: []string{\"unreadable record: \" + reason}}\n13\t}"},{"item":"ran `grep -n Incomplete src/audit.go`","kind":"probe","round":1,"sha256":"","text":"exit status: 0\ntimed_out: false\nwall: <wall> s\n10:// Incomplete is the observation of a parcel whose record could not be read: safe diagnostics only.\n11:func Incomplete(parcel, reason string) Observation {\n"}]},
        {"id":"D2","round":1,"decision":"Normal parcel reads keep their current behavior","constraints":[],"uncertainty":"the repository has no test of normal reads to cite","revises":"","why":"","superseded_by":"","evidence":[]}]}
      """
    And k-shape-sourced-decision-continuity: the observation "question_check_input_1" is exactly:
      """
      Slug: is-even

      # Request

      Make parcel inspection obey the existing safety policy; preserve normal reads. The new audit panel's visual presentation is open.


      # Earlier questions and answers

      none

      # Questions

      1. What should a user experience on an unreadable record: reject the operation or save an incomplete observation?
         a) Save an incomplete observation (recommended)
         b) Reject the operation
      """
    And k-shape-sourced-decision-continuity: the observation "question_check_instructions" is exactly:
      """
      You check the questions a shaper wants to ask the user about one feature request. Only UI/UX choices that the request and the user's answers leave open, and mandatory authentication or security handoffs, may reach the user; the shaper investigates and decides everything else itself. You see only the request text, the earlier questions with the user's answers, the decisions this shaping recorded with the evidence they cite, the questions, and the cited evidence: the text this shaping read or ran for each (evidence: ...) item a question cites. Read-only: do not use any tool except submit_question_check.

      Give one item for every question: its number, one class, why (one short reason) and decision (the id of the recorded decision that settles it when the class is decided, otherwise an empty string). The classes:
      - ui_ux: a choice of what the user sees or does in the product's interface or interaction (layout, wording, flow, feedback, input behavior) that the request and the earlier answers leave open;
      - handoff: a mandatory authentication or security step that only the user can perform, such as signing in or granting access;
      - answered: the request or an earlier answer already settles it;
      - decided: a recorded decision settles it, or settles the technical, security, protocol, storage or limit choice it asks about, whatever its wording or the granularity of its options;
      - technical: an implementation, architecture, library, protocol, data, storage, limit, security or testing choice, even when it is worded as what the user experiences or chooses;
      - product_policy: a business, pricing, permission or other product rule that is not a choice of interface or interaction;
      - environment: whether a tool, service, account, credential, network or other resource is available where the work runs, when it is not an environment_prerequisite;
      - environment_prerequisite: a resource without which this shaping cannot write the Intent or run its acceptance, as the question's cited evidence shows: the cited text is about the resource the question names and shows it needed and missing, and neither the cited text, the request nor the earlier answers show it optional, needed only to run the finished feature against a real external service, or already stood in for by a fake, stub, simulation or recorded response; the why names the part of the Intent or acceptance that cannot be written or run without it and what in the cited text shows that. A question with no cited text shown, or whose cited text does not support the claimed necessity, is environment, whatever its own wording claims;
      - repository_fact: a fact that the repository's files, history or commands can establish.
      Answer by calling submit_question_check.
      """
    And k-shape-sourced-decision-continuity: the observation "question_check_schema" is this JSON:
      """
      {"type": "object",
       "properties": {"items":{"type":"array","items":{"type":"object","properties":{"number":{"type":"integer"},"class":{"type":"string","enum":["ui_ux","handoff","answered","decided","technical","product_policy","environment","environment_prerequisite","repository_fact"]},"why":{"type":"string"},"decision":{"type":"string"}},"required":["number","class","why","decision"],"additionalProperties":false}}},
       "required": ["items"],
       "additionalProperties": false}
      """

  Scenario: The open presentation question is still asked with its options and recommendation, and the recorded decision is retained while shaping stops
    When k-shape-sourced-decision-continuity: the "settle" journey is shaped
    Then k-shape-sourced-decision-continuity: the observations include:
      """
      {"ledger_mode_1": "0600",
       "question_check_classes": [
        ["technical"],
        ["ui_ux"],
        ["decided"]]}
      """
    And k-shape-sourced-decision-continuity: the observation "stdout_1" is exactly these lines:
      """
      Questions for is-even:
      1. Show audit details inline or in an expandable panel?
         a) In an expandable panel (recommended)
         b) Inline
      Answer one per line as <n>: <letter or your words>, e.g. printf '1: a\n' | kogen intent shape is-even -
      Or enter all: recommended to use every question's recommended option.
      """
    And k-shape-sourced-decision-continuity: the observation "question_check_input_2" is exactly:
      """
      Slug: is-even

      # Request

      Make parcel inspection obey the existing safety policy; preserve normal reads. The new audit panel's visual presentation is open.


      # Earlier questions and answers

      none

      # Recorded decisions

      D1 (round 1): An unreadable parcel record produces an incomplete audit observation with safe diagnostics, built with audit.Incomplete
         constraint: record bytes are never included
         evidence: docs/policy.md (read by this shaping in round 1)
         evidence: src/audit.go (read by this shaping in round 1)
         evidence: ran `grep -n Incomplete src/audit.go` (probe output in round 1)

      D1 cites docs/policy.md (read by this shaping in round 1):
      1	# Parcel safety policy
      2	
      3	An unreadable parcel record produces an incomplete audit observation with safe diagnostics; record bytes are never included.

      D1 cites src/audit.go (read by this shaping in round 1):
      1	package audit
      2	
      3	// Observation is one audit observation of a parcel inspection.
      4	type Observation struct {
      5		Parcel      string
      6		Complete    bool
      7		Diagnostics []string
      8	}
      9	
      10	// Incomplete is the observation of a parcel whose record could not be read: safe diagnostics only.
      11	func Incomplete(parcel, reason string) Observation {
      12		return Observation{Parcel: parcel, Diagnostics: []string{"unreadable record: " + reason}}
      13	}

      D1 cites ran `grep -n Incomplete src/audit.go` (probe output in round 1):
      exit status: 0
      timed_out: false
      wall: <wall> s
      10:// Incomplete is the observation of a parcel whose record could not be read: safe diagnostics only.
      11:func Incomplete(parcel, reason string) Observation {

      D2 (round 1): Normal parcel reads keep their current behavior
         uncertainty: the repository has no test of normal reads to cite

      # Questions

      1. Show audit details inline or in an expandable panel?
         a) In an expandable panel (recommended)
         b) Inline
      """

  Scenario: After the answer, a subchoice the recorded decision settles is not asked, the Intent must carry the decision and its constraint, and the first round's receipts stand in for it
    When k-shape-sourced-decision-continuity: the "settle" journey is shaped
    Then k-shape-sourced-decision-continuity: the observations include:
      """
      {"exits": [5,0],
       "errors": ["",""],
       "last_stdout_lines": ["Or enter all: recommended to use every question's recommended option.","Next: kogen intent approve is-even"],
       "shaper_requests": [4,3],
       "question_check_requests": [2,1],
       "refused_gates": ["question_check","decision_record","question_check","carried_decisions"],
       "question_check_decisions": [
        [null],
        [null],
        ["D1"]],
       "decisions_recorded": [
        {"round":1,"recorded":["D1","D2"],"kept":[]},
        {"round":2,"recorded":[],"kept":["D1","D2"]}],
       "carried_evidence": [
        [{"decision":1,"item":"docs/policy.md","kind":"read","round":1},{"decision":1,"item":"src/audit.go","kind":"read","round":1},{"decision":1,"item":"ran `grep -n Incomplete src/audit.go`","kind":"probe","round":1}]],
       "ledger_unchanged_2": true,
       "intent_exists": true,
       "published": true}
      """
    And k-shape-sourced-decision-continuity: the observation "ledger_2" is this JSON:
      """
      {"version": 1,
       "decisions": [
        {"id":"D1","round":1,"decision":"An unreadable parcel record produces an incomplete audit observation with safe diagnostics, built with audit.Incomplete","constraints":["record bytes are never included"],"uncertainty":"","revises":"","why":"","superseded_by":"","evidence":[{"item":"docs/policy.md","kind":"read","round":1,"sha256":"6f4f4c3e299d5b91551b2483f35694775247fafc849bfb711c8a631961b70f9a","text":"1\t# Parcel safety policy\n2\t\n3\tAn unreadable parcel record produces an incomplete audit observation with safe diagnostics; record bytes are never included."},{"item":"src/audit.go","kind":"read","round":1,"sha256":"85896eb548276f1995490420465ce5a15b659f2c55b9658c3ea09cda018bbfa7","text":"1\tpackage audit\n2\t\n3\t// Observation is one audit observation of a parcel inspection.\n4\ttype Observation struct {\n5\t\tParcel      string\n6\t\tComplete    bool\n7\t\tDiagnostics []string\n8\t}\n9\t\n10\t// Incomplete is the observation of a parcel whose record could not be read: safe diagnostics only.\n11\tfunc Incomplete(parcel, reason string) Observation {\n12\t\treturn Observation{Parcel: parcel, Diagnostics: []string{\"unreadable record: \" + reason}}\n13\t}"},{"item":"ran `grep -n Incomplete src/audit.go`","kind":"probe","round":1,"sha256":"","text":"exit status: 0\ntimed_out: false\nwall: <wall> s\n10:// Incomplete is the observation of a parcel whose record could not be read: safe diagnostics only.\n11:func Incomplete(parcel, reason string) Observation {\n"}]},
        {"id":"D2","round":1,"decision":"Normal parcel reads keep their current behavior","constraints":[],"uncertainty":"the repository has no test of normal reads to cite","revises":"","why":"","superseded_by":"","evidence":[]}]}
      """
    And k-shape-sourced-decision-continuity: the observation "context_decisions_2" is exactly:
      """
      # Your recorded decisions (each stands until a revision that cites evidence this round read or ran replaces it; keep every one that stands in the Intent's Decisions exactly as its Intent line shows)

      D1 (round 1): An unreadable parcel record produces an incomplete audit observation with safe diagnostics, built with audit.Incomplete
         constraint: record bytes are never included
         evidence: docs/policy.md (read by this shaping in round 1)
         evidence: src/audit.go (read by this shaping in round 1)
         evidence: ran `grep -n Incomplete src/audit.go` (probe output in round 1)
         Intent line: - An unreadable parcel record produces an incomplete audit observation with safe diagnostics, built with audit.Incomplete (decision: D1, round 1) (constraint: record bytes are never included) (evidence: docs/policy.md; src/audit.go; ran `grep -n Incomplete src/audit.go`) (source: request)

      D2 (round 1): Normal parcel reads keep their current behavior
         uncertainty: the repository has no test of normal reads to cite
         Intent line: - Normal parcel reads keep their current behavior (decision: D2, round 1) (assumption: the repository has no test of normal reads to cite) (source: request)
      """
    And k-shape-sourced-decision-continuity: the observation "question_check_input_3" is exactly:
      """
      Slug: is-even

      # Request

      Make parcel inspection obey the existing safety policy; preserve normal reads. The new audit panel's visual presentation is open.


      # Earlier questions and answers

      Round 1:
      1. Show audit details inline or in an expandable panel?
         a) In an expandable panel
         b) Inline
         answer: 1: a

      # Recorded decisions

      D1 (round 1): An unreadable parcel record produces an incomplete audit observation with safe diagnostics, built with audit.Incomplete
         constraint: record bytes are never included
         evidence: docs/policy.md (read by this shaping in round 1)
         evidence: src/audit.go (read by this shaping in round 1)
         evidence: ran `grep -n Incomplete src/audit.go` (probe output in round 1)

      D1 cites docs/policy.md (read by this shaping in round 1):
      1	# Parcel safety policy
      2	
      3	An unreadable parcel record produces an incomplete audit observation with safe diagnostics; record bytes are never included.

      D1 cites src/audit.go (read by this shaping in round 1):
      1	package audit
      2	
      3	// Observation is one audit observation of a parcel inspection.
      4	type Observation struct {
      5		Parcel      string
      6		Complete    bool
      7		Diagnostics []string
      8	}
      9	
      10	// Incomplete is the observation of a parcel whose record could not be read: safe diagnostics only.
      11	func Incomplete(parcel, reason string) Observation {
      12		return Observation{Parcel: parcel, Diagnostics: []string{"unreadable record: " + reason}}
      13	}

      D1 cites ran `grep -n Incomplete src/audit.go` (probe output in round 1):
      exit status: 0
      timed_out: false
      wall: <wall> s
      10:// Incomplete is the observation of a parcel whose record could not be read: safe diagnostics only.
      11:func Incomplete(parcel, reason string) Observation {

      D2 (round 1): Normal parcel reads keep their current behavior
         uncertainty: the repository has no test of normal reads to cite

      # Questions

      1. For incomplete parcel observations, should record bytes be attached or only safe diagnostics?
         a) Only safe diagnostics (recommended)
         b) Attach record bytes
      """
    And k-shape-sourced-decision-continuity: the observation "feedback_s4" is exactly:
      """
      Question check: 1 of 1 question(s) are not UI/UX choices left to the user; nothing was asked. Investigate and decide each one yourself: read the repository or run a probe, then record a Decisions line that cites what you read or ran before its source tag, as - <decision> (evidence: <repository path>) (source: request) or - <decision> (evidence: ran `<command>`) (source: request). Where the repository has no evidence or conflicting evidence, record the uncertainty you decided under instead, as - <decision> (assumption: <the uncertainty>) (source: request); the user confirms or reverses it at approval. Ask the user only UI/UX questions that the request and the answers leave open, or a mandatory authentication or security handoff.
      - question 1 (decided): For incomplete parcel observations, should record bytes be attached or only safe diagnostics? | why: D1 settles it: record bytes are never included | D1 (round 1) settles it: An unreadable parcel record produces an incomplete audit observation with safe diagnostics, built with audit.Incomplete; do not ask it again. Complete the choice from D1, its constraints and its evidence, and keep it in the Intent's Decisions as: - An unreadable parcel record produces an incomplete audit observation with safe diagnostics, built with audit.Incomplete (decision: D1, round 1) (constraint: record bytes are never included) (evidence: docs/policy.md; src/audit.go; ran `grep -n Incomplete src/audit.go`) (source: request)
      """
    And k-shape-sourced-decision-continuity: the observation "feedback_s5" is exactly:
      """
      Recorded decisions: 2 recorded decision(s) that stand are missing from the Intent's Decisions; nothing was published. A recorded decision stands until a revision that cites evidence this round read or ran replaces it, and its line is kept exactly, before its source tag, so its evidence and constraints stay in force:
      - D1 (round 1) is not in Decisions; keep it as: - An unreadable parcel record produces an incomplete audit observation with safe diagnostics, built with audit.Incomplete (decision: D1, round 1) (constraint: record bytes are never included) (evidence: docs/policy.md; src/audit.go; ran `grep -n Incomplete src/audit.go`) (source: request)
      - D2 (round 1) is not in Decisions; keep it as: - Normal parcel reads keep their current behavior (decision: D2, round 1) (assumption: the repository has no test of normal reads to cite) (source: request)
      """

  Scenario: On a further continuation a restated settled question is not asked, and a revision needs the changed policy actually read and keeps the old decision on record
    When k-shape-sourced-decision-continuity: the "revise" journey is shaped
    Then k-shape-sourced-decision-continuity: the observations include:
      """
      {"exits": [5,5,0],
       "errors": ["","",""],
       "shaper_requests": [4,1,4],
       "question_check_requests": [2,1,1],
       "refused_gates": ["question_check","decision_record","question_check","decision_record"],
       "question_check_classes": [
        ["technical"],
        ["ui_ux"],
        ["ui_ux"],
        ["decided"]],
       "question_check_decisions": [
        [null],
        [null],
        [null],
        ["D1"]],
       "decisions_recorded": [
        {"round":1,"recorded":["D1","D2"],"kept":[]},
        {"round":3,"recorded":["D3"],"kept":[]}],
       "carried_evidence": [
        [{"decision":1,"item":"src/audit.go","kind":"read","round":1}]],
       "ledger_unchanged_2": true,
       "ledger_unchanged_3": false,
       "intent_exists": true,
       "published": true}
      """
    And k-shape-sourced-decision-continuity: the observation "stdout_2" is exactly these lines:
      """
      Questions for is-even:
      1. Should the audit panel list the newest observation first or the oldest first?
         a) Newest first (recommended)
         b) Oldest first
      Answer one per line as <n>: <letter or your words>, e.g. printf '1: a\n' | kogen intent shape is-even -
      Or enter all: recommended to use every question's recommended option.
      """
    And k-shape-sourced-decision-continuity: the observation "context_decisions_3" is exactly:
      """
      # Your recorded decisions (each stands until a revision that cites evidence this round read or ran replaces it; keep every one that stands in the Intent's Decisions exactly as its Intent line shows)

      D1 (round 1): An unreadable parcel record produces an incomplete audit observation with safe diagnostics, built with audit.Incomplete
         constraint: record bytes are never included
         evidence: docs/policy.md (read by this shaping in round 1; changed since)
         evidence: src/audit.go (read by this shaping in round 1)
         evidence: ran `grep -n Incomplete src/audit.go` (probe output in round 1)
         Intent line: - An unreadable parcel record produces an incomplete audit observation with safe diagnostics, built with audit.Incomplete (decision: D1, round 1) (constraint: record bytes are never included) (evidence: docs/policy.md; src/audit.go; ran `grep -n Incomplete src/audit.go`) (source: request)

      D2 (round 1): Normal parcel reads keep their current behavior
         uncertainty: the repository has no test of normal reads to cite
         Intent line: - Normal parcel reads keep their current behavior (decision: D2, round 1) (assumption: the repository has no test of normal reads to cite) (source: request)
      """
    And k-shape-sourced-decision-continuity: the observation "question_check_input_4" is exactly:
      """
      Slug: is-even

      # Request

      Make parcel inspection obey the existing safety policy; preserve normal reads. The new audit panel's visual presentation is open.


      # Earlier questions and answers

      Round 1:
      1. Show audit details inline or in an expandable panel?
         a) In an expandable panel
         b) Inline
         answer: 1: a
      Round 2:
      1. Should the audit panel list the newest observation first or the oldest first?
         a) Newest first
         b) Oldest first
         answer: 1: a

      # Recorded decisions

      D1 (round 1): An unreadable parcel record produces an incomplete audit observation with safe diagnostics, built with audit.Incomplete
         constraint: record bytes are never included
         evidence: docs/policy.md (read by this shaping in round 1; changed since)
         evidence: src/audit.go (read by this shaping in round 1)
         evidence: ran `grep -n Incomplete src/audit.go` (probe output in round 1)

      D1 cites docs/policy.md (read by this shaping in round 1; changed since):
      1	# Parcel safety policy
      2	
      3	An unreadable parcel record produces an incomplete audit observation with safe diagnostics; record bytes are never included.

      D1 cites src/audit.go (read by this shaping in round 1):
      1	package audit
      2	
      3	// Observation is one audit observation of a parcel inspection.
      4	type Observation struct {
      5		Parcel      string
      6		Complete    bool
      7		Diagnostics []string
      8	}
      9	
      10	// Incomplete is the observation of a parcel whose record could not be read: safe diagnostics only.
      11	func Incomplete(parcel, reason string) Observation {
      12		return Observation{Parcel: parcel, Diagnostics: []string{"unreadable record: " + reason}}
      13	}

      D1 cites ran `grep -n Incomplete src/audit.go` (probe output in round 1):
      exit status: 0
      timed_out: false
      wall: <wall> s
      10:// Incomplete is the observation of a parcel whose record could not be read: safe diagnostics only.
      11:func Incomplete(parcel, reason string) Observation {

      D2 (round 1): Normal parcel reads keep their current behavior
         uncertainty: the repository has no test of normal reads to cite

      # Questions

      1. When a parcel record cannot be read, should the saved observation carry the raw record or just its diagnostics?
         a) Just its diagnostics (recommended)
         b) The raw record
      """
    And k-shape-sourced-decision-continuity: the observation "feedback_s5" is exactly:
      """
      Question check: 1 of 1 question(s) are not UI/UX choices left to the user; nothing was asked. Investigate and decide each one yourself: read the repository or run a probe, then record a Decisions line that cites what you read or ran before its source tag, as - <decision> (evidence: <repository path>) (source: request) or - <decision> (evidence: ran `<command>`) (source: request). Where the repository has no evidence or conflicting evidence, record the uncertainty you decided under instead, as - <decision> (assumption: <the uncertainty>) (source: request); the user confirms or reverses it at approval. Ask the user only UI/UX questions that the request and the answers leave open, or a mandatory authentication or security handoff.
      - question 1 (decided): When a parcel record cannot be read, should the saved observation carry the raw record or just its diagnostics? | why: D1 settles it in other words: an incomplete observation carries safe diagnostics, never the record | D1 (round 1) settles it: An unreadable parcel record produces an incomplete audit observation with safe diagnostics, built with audit.Incomplete; do not ask it again. Complete the choice from D1, its constraints and its evidence, and keep it in the Intent's Decisions as: - An unreadable parcel record produces an incomplete audit observation with safe diagnostics, built with audit.Incomplete (decision: D1, round 1) (constraint: record bytes are never included) (evidence: docs/policy.md; src/audit.go; ran `grep -n Incomplete src/audit.go`) (source: request)
      """
    And k-shape-sourced-decision-continuity: the observation "feedback_s6" is exactly:
      """
      Sourced decisions: 2 problem(s) with the decisions this submission records; nothing was recorded, asked or published. A sourced decision cites, as evidence items, files this shaping read (or that an earlier round read and that are unchanged since) or ran `<command>` for a probe of this shaping that succeeded; where there is no such evidence, state the uncertainty you decided under instead. A revision names the decision it revises in revises, says why it changed and cites evidence this round read or ran.
      - sourced decision 1: docs/policy.md changed since round 1 read it; read it again
      - sourced decision 1: revises D1 without evidence this round read or ran, so D1 stands
      """
    And k-shape-sourced-decision-continuity: the observation "ledger_3" is this JSON:
      """
      {"version": 1,
       "decisions": [
        {"id":"D1","round":1,"decision":"An unreadable parcel record produces an incomplete audit observation with safe diagnostics, built with audit.Incomplete","constraints":["record bytes are never included"],"uncertainty":"","revises":"","why":"","superseded_by":"D3","evidence":[{"item":"docs/policy.md","kind":"read","round":1,"sha256":"6f4f4c3e299d5b91551b2483f35694775247fafc849bfb711c8a631961b70f9a","text":"1\t# Parcel safety policy\n2\t\n3\tAn unreadable parcel record produces an incomplete audit observation with safe diagnostics; record bytes are never included."},{"item":"src/audit.go","kind":"read","round":1,"sha256":"85896eb548276f1995490420465ce5a15b659f2c55b9658c3ea09cda018bbfa7","text":"1\tpackage audit\n2\t\n3\t// Observation is one audit observation of a parcel inspection.\n4\ttype Observation struct {\n5\t\tParcel      string\n6\t\tComplete    bool\n7\t\tDiagnostics []string\n8\t}\n9\t\n10\t// Incomplete is the observation of a parcel whose record could not be read: safe diagnostics only.\n11\tfunc Incomplete(parcel, reason string) Observation {\n12\t\treturn Observation{Parcel: parcel, Diagnostics: []string{\"unreadable record: \" + reason}}\n13\t}"},{"item":"ran `grep -n Incomplete src/audit.go`","kind":"probe","round":1,"sha256":"","text":"exit status: 0\ntimed_out: false\nwall: <wall> s\n10:// Incomplete is the observation of a parcel whose record could not be read: safe diagnostics only.\n11:func Incomplete(parcel, reason string) Observation {\n"}]},
        {"id":"D2","round":1,"decision":"Normal parcel reads keep their current behavior","constraints":[],"uncertainty":"the repository has no test of normal reads to cite","revises":"","why":"","superseded_by":"","evidence":[]},
        {"id":"D3","round":3,"decision":"An unreadable parcel record produces an incomplete audit observation with safe diagnostics and the record's size, built with audit.Incomplete","constraints":["record bytes are never included"],"uncertainty":"","revises":"D1","why":"docs/policy.md now also requires the record's size in the diagnostics","superseded_by":"","evidence":[{"item":"docs/policy.md","kind":"read","round":3,"sha256":"845603cda93363afb07b90a39a8b95c33e358bd19f30520164c67492d9c8c72e","text":"1\t# Parcel safety policy\n2\t\n3\tAn unreadable parcel record produces an incomplete audit observation with safe diagnostics and the record's size; record bytes are never included."},{"item":"src/audit.go","kind":"read","round":1,"sha256":"85896eb548276f1995490420465ce5a15b659f2c55b9658c3ea09cda018bbfa7","text":"1\tpackage audit\n2\t\n3\t// Observation is one audit observation of a parcel inspection.\n4\ttype Observation struct {\n5\t\tParcel      string\n6\t\tComplete    bool\n7\t\tDiagnostics []string\n8\t}\n9\t\n10\t// Incomplete is the observation of a parcel whose record could not be read: safe diagnostics only.\n11\tfunc Incomplete(parcel, reason string) Observation {\n12\t\treturn Observation{Parcel: parcel, Diagnostics: []string{\"unreadable record: \" + reason}}\n13\t}"}]}]}
      """

  Scenario: A shaper that keeps proposing a settled question stops at the existing submission limit with the cause retained, asking nothing
    When k-shape-sourced-decision-continuity: the "repeat" journey is shaped
    Then k-shape-sourced-decision-continuity: the observations include:
      """
      {"exits": [5,1],
       "errors": ["","shape/invalid_output: the shaper's output was not a valid Intent after 3 attempts"],
       "error_lines": ["- question 1 (decided): For incomplete parcel observations, should record bytes be attached or only safe diagnostics? | why: D1 settles it: record bytes are never included | D1 (round 1) settles it: An unreadable parcel record produces an incomplete audit observation with safe diagnostics, built with audit.Incomplete; do not ask it again. Complete the choice from D1, its constraints and its evidence, and keep it in the Intent's Decisions as: - An unreadable parcel record produces an incomplete audit observation with safe diagnostics, built with audit.Incomplete (decision: D1, round 1) (constraint: record bytes are never included) (evidence: docs/policy.md; src/audit.go; ran `grep -n Incomplete src/audit.go`) (source: request)","question check refusals: decided x3"],
       "last_stdout_lines": ["Or enter all: recommended to use every question's recommended option.",""],
       "shaper_requests": [4,3],
       "question_check_requests": [2,3],
       "refused_gates": ["question_check","decision_record","question_check","question_check","question_check"],
       "question_check_classes": [
        ["technical"],
        ["ui_ux"],
        ["decided"],
        ["decided"],
        ["decided"]],
       "question_check_decisions": [
        [null],
        [null],
        ["D1"],
        ["D1"],
        ["D1"]],
       "decisions_recorded": [
        {"round":1,"recorded":["D1","D2"],"kept":[]}],
       "ledger_unchanged_2": true,
       "intent_exists": false,
       "published": false}
      """
    And k-shape-sourced-decision-continuity: the observation "ledger_2" is this JSON:
      """
      {"version": 1,
       "decisions": [
        {"id":"D1","round":1,"decision":"An unreadable parcel record produces an incomplete audit observation with safe diagnostics, built with audit.Incomplete","constraints":["record bytes are never included"],"uncertainty":"","revises":"","why":"","superseded_by":"","evidence":[{"item":"docs/policy.md","kind":"read","round":1,"sha256":"6f4f4c3e299d5b91551b2483f35694775247fafc849bfb711c8a631961b70f9a","text":"1\t# Parcel safety policy\n2\t\n3\tAn unreadable parcel record produces an incomplete audit observation with safe diagnostics; record bytes are never included."},{"item":"src/audit.go","kind":"read","round":1,"sha256":"85896eb548276f1995490420465ce5a15b659f2c55b9658c3ea09cda018bbfa7","text":"1\tpackage audit\n2\t\n3\t// Observation is one audit observation of a parcel inspection.\n4\ttype Observation struct {\n5\t\tParcel      string\n6\t\tComplete    bool\n7\t\tDiagnostics []string\n8\t}\n9\t\n10\t// Incomplete is the observation of a parcel whose record could not be read: safe diagnostics only.\n11\tfunc Incomplete(parcel, reason string) Observation {\n12\t\treturn Observation{Parcel: parcel, Diagnostics: []string{\"unreadable record: \" + reason}}\n13\t}"},{"item":"ran `grep -n Incomplete src/audit.go`","kind":"probe","round":1,"sha256":"","text":"exit status: 0\ntimed_out: false\nwall: <wall> s\n10:// Incomplete is the observation of a parcel whose record could not be read: safe diagnostics only.\n11:func Incomplete(parcel, reason string) Observation {\n"}]},
        {"id":"D2","round":1,"decision":"Normal parcel reads keep their current behavior","constraints":[],"uncertainty":"the repository has no test of normal reads to cite","revises":"","why":"","superseded_by":"","evidence":[]}]}
      """

  Scenario: The external shaper records, carries and settles decisions the same way, and its command output is no evidence
    When k-shape-sourced-decision-continuity: the "external" journey is shaped
    Then k-shape-sourced-decision-continuity: the observations include:
      """
      {"exits": [5,0],
       "errors": ["",""],
       "shaper_requests": [3,2],
       "question_check_requests": [2,1],
       "refused_gates": ["question_check","decision_record","question_check"],
       "question_check_classes": [
        ["technical"],
        ["ui_ux"],
        ["decided"]],
       "question_check_decisions": [
        [null],
        [null],
        ["D1"]],
       "decisions_recorded": [
        {"round":1,"recorded":["D1","D2"],"kept":[]}],
       "carried_evidence": [
        [{"decision":1,"item":"docs/policy.md","kind":"checkout","round":1},{"decision":1,"item":"src/audit.go","kind":"checkout","round":1}]],
       "ledger_mode_1": "0600",
       "ledger_unchanged_2": true,
       "intent_exists": true,
       "published": true}
      """
    And k-shape-sourced-decision-continuity: the observation "feedback_x1" is exactly:
      """
      Question check: 1 of 1 question(s) are not UI/UX choices left to the user; nothing was asked. Investigate and decide each one yourself: read the repository or run a probe, then record a Decisions line that cites what you read or ran before its source tag, as - <decision> (evidence: <repository path>) (source: request) or - <decision> (evidence: ran `<command>`) (source: request). Where the repository has no evidence or conflicting evidence, record the uncertainty you decided under instead, as - <decision> (assumption: <the uncertainty>) (source: request); the user confirms or reverses it at approval. Ask the user only UI/UX questions that the request and the answers leave open, or a mandatory authentication or security handoff.
      - question 1 (technical): What should a user experience on an unreadable record: reject the operation or save an incomplete observation? | why: rejecting or saving an unreadable record is a security and data-handling choice, whatever its wording | decide it from what the repository and the request establish
      """
    And k-shape-sourced-decision-continuity: the observation "feedback_x2" is exactly:
      """
      Sourced decisions: 1 problem(s) with the decisions this submission records; nothing was recorded, asked or published. A sourced decision cites, as evidence items, files this shaping read (or that an earlier round read and that are unchanged since) or ran `<command>` for a probe of this shaping that succeeded; where there is no such evidence, state the uncertainty you decided under instead. A revision names the decision it revises in revises, says why it changed and cites evidence this round read or ran.
      - sourced decision 1: Kogen keeps no output of the external session's commands, so `grep -n Incomplete src/audit.go` is no evidence; cite a file this session read
      """
    And k-shape-sourced-decision-continuity: the observation "ledger_1" is this JSON:
      """
      {"version": 1,
       "decisions": [
        {"id":"D1","round":1,"decision":"An unreadable parcel record produces an incomplete audit observation with safe diagnostics, built with audit.Incomplete","constraints":["record bytes are never included"],"uncertainty":"","revises":"","why":"","superseded_by":"","evidence":[{"item":"docs/policy.md","kind":"checkout","round":1,"sha256":"6f4f4c3e299d5b91551b2483f35694775247fafc849bfb711c8a631961b70f9a","text":"1\t# Parcel safety policy\n2\t\n3\tAn unreadable parcel record produces an incomplete audit observation with safe diagnostics; record bytes are never included."},{"item":"src/audit.go","kind":"checkout","round":1,"sha256":"85896eb548276f1995490420465ce5a15b659f2c55b9658c3ea09cda018bbfa7","text":"1\tpackage audit\n2\t\n3\t// Observation is one audit observation of a parcel inspection.\n4\ttype Observation struct {\n5\t\tParcel      string\n6\t\tComplete    bool\n7\t\tDiagnostics []string\n8\t}\n9\t\n10\t// Incomplete is the observation of a parcel whose record could not be read: safe diagnostics only.\n11\tfunc Incomplete(parcel, reason string) Observation {\n12\t\treturn Observation{Parcel: parcel, Diagnostics: []string{\"unreadable record: \" + reason}}\n13\t}"}]},
        {"id":"D2","round":1,"decision":"Normal parcel reads keep their current behavior","constraints":[],"uncertainty":"the repository has no test of normal reads to cite","revises":"","why":"","superseded_by":"","evidence":[]}]}
      """
    And k-shape-sourced-decision-continuity: the observation "question_check_input_2" is exactly:
      """
      Slug: is-even

      # Request

      Make parcel inspection obey the existing safety policy; preserve normal reads. The new audit panel's visual presentation is open.


      # Earlier questions and answers

      none

      # Recorded decisions

      D1 (round 1): An unreadable parcel record produces an incomplete audit observation with safe diagnostics, built with audit.Incomplete
         constraint: record bytes are never included
         evidence: docs/policy.md (read from the checkout in round 1)
         evidence: src/audit.go (read from the checkout in round 1)

      D1 cites docs/policy.md (read from the checkout in round 1):
      1	# Parcel safety policy
      2	
      3	An unreadable parcel record produces an incomplete audit observation with safe diagnostics; record bytes are never included.

      D1 cites src/audit.go (read from the checkout in round 1):
      1	package audit
      2	
      3	// Observation is one audit observation of a parcel inspection.
      4	type Observation struct {
      5		Parcel      string
      6		Complete    bool
      7		Diagnostics []string
      8	}
      9	
      10	// Incomplete is the observation of a parcel whose record could not be read: safe diagnostics only.
      11	func Incomplete(parcel, reason string) Observation {
      12		return Observation{Parcel: parcel, Diagnostics: []string{"unreadable record: " + reason}}
      13	}

      D2 (round 1): Normal parcel reads keep their current behavior
         uncertainty: the repository has no test of normal reads to cite

      # Questions

      1. Show audit details inline or in an expandable panel?
         a) In an expandable panel (recommended)
         b) Inline
      """
    And k-shape-sourced-decision-continuity: the observation "context_decisions_2" is exactly:
      """
      # Your recorded decisions (each stands until a revision that cites evidence this round read or ran replaces it; keep every one that stands in the Intent's Decisions exactly as its Intent line shows)

      D1 (round 1): An unreadable parcel record produces an incomplete audit observation with safe diagnostics, built with audit.Incomplete
         constraint: record bytes are never included
         evidence: docs/policy.md (read from the checkout in round 1)
         evidence: src/audit.go (read from the checkout in round 1)
         Intent line: - An unreadable parcel record produces an incomplete audit observation with safe diagnostics, built with audit.Incomplete (decision: D1, round 1) (constraint: record bytes are never included) (evidence: docs/policy.md; src/audit.go) (source: request)

      D2 (round 1): Normal parcel reads keep their current behavior
         uncertainty: the repository has no test of normal reads to cite
         Intent line: - Normal parcel reads keep their current behavior (decision: D2, round 1) (assumption: the repository has no test of normal reads to cite) (source: request)
      """
    And k-shape-sourced-decision-continuity: the observation "question_check_input_3" is exactly:
      """
      Slug: is-even

      # Request

      Make parcel inspection obey the existing safety policy; preserve normal reads. The new audit panel's visual presentation is open.


      # Earlier questions and answers

      Round 1:
      1. Show audit details inline or in an expandable panel?
         a) In an expandable panel
         b) Inline
         answer: 1: a

      # Recorded decisions

      D1 (round 1): An unreadable parcel record produces an incomplete audit observation with safe diagnostics, built with audit.Incomplete
         constraint: record bytes are never included
         evidence: docs/policy.md (read from the checkout in round 1)
         evidence: src/audit.go (read from the checkout in round 1)

      D1 cites docs/policy.md (read from the checkout in round 1):
      1	# Parcel safety policy
      2	
      3	An unreadable parcel record produces an incomplete audit observation with safe diagnostics; record bytes are never included.

      D1 cites src/audit.go (read from the checkout in round 1):
      1	package audit
      2	
      3	// Observation is one audit observation of a parcel inspection.
      4	type Observation struct {
      5		Parcel      string
      6		Complete    bool
      7		Diagnostics []string
      8	}
      9	
      10	// Incomplete is the observation of a parcel whose record could not be read: safe diagnostics only.
      11	func Incomplete(parcel, reason string) Observation {
      12		return Observation{Parcel: parcel, Diagnostics: []string{"unreadable record: " + reason}}
      13	}

      D2 (round 1): Normal parcel reads keep their current behavior
         uncertainty: the repository has no test of normal reads to cite

      # Questions

      1. For incomplete parcel observations, should record bytes be attached or only safe diagnostics?
         a) Only safe diagnostics (recommended)
         b) Attach record bytes
      """
    And k-shape-sourced-decision-continuity: the observation "feedback_x4" is exactly:
      """
      Question check: 1 of 1 question(s) are not UI/UX choices left to the user; nothing was asked. Investigate and decide each one yourself: read the repository or run a probe, then record a Decisions line that cites what you read or ran before its source tag, as - <decision> (evidence: <repository path>) (source: request) or - <decision> (evidence: ran `<command>`) (source: request). Where the repository has no evidence or conflicting evidence, record the uncertainty you decided under instead, as - <decision> (assumption: <the uncertainty>) (source: request); the user confirms or reverses it at approval. Ask the user only UI/UX questions that the request and the answers leave open, or a mandatory authentication or security handoff.
      - question 1 (decided): For incomplete parcel observations, should record bytes be attached or only safe diagnostics? | why: D1 settles it: record bytes are never included | D1 (round 1) settles it: An unreadable parcel record produces an incomplete audit observation with safe diagnostics, built with audit.Incomplete; do not ask it again. Complete the choice from D1, its constraints and its evidence, and keep it in the Intent's Decisions as: - An unreadable parcel record produces an incomplete audit observation with safe diagnostics, built with audit.Incomplete (decision: D1, round 1) (constraint: record bytes are never included) (evidence: docs/policy.md; src/audit.go) (source: request)
      """
    And k-shape-sourced-decision-continuity: the observation "question_check_schema" is this JSON:
      """
      {"type": "object",
       "properties": {"items":{"type":"array","items":{"type":"object","properties":{"number":{"type":"integer"},"class":{"type":"string","enum":["ui_ux","handoff","answered","decided","technical","product_policy","environment","environment_prerequisite","repository_fact"]},"why":{"type":"string"},"decision":{"type":"string"}},"required":["number","class","why","decision"],"additionalProperties":false}}},
       "required": ["items"],
       "additionalProperties": false}
      """

  Scenario: An ordinary first-pass Intent keeps its normal path, with no decision ledger and no new gate
    When k-shape-sourced-decision-continuity: the "ordinary" journey is shaped
    Then k-shape-sourced-decision-continuity: the observations include:
      """
      {"exits": [0],
       "errors": [""],
       "error_lines": [],
       "last_stdout_lines": ["Next: kogen intent approve is-even"],
       "shaper_requests": [1],
       "question_check_requests": [0],
       "refused_gates": [],
       "question_check_classes": [],
       "decisions_recorded": [],
       "carried_evidence": [],
       "ledger_1": null,
       "context_decisions_1": "",
       "intent_exists": true,
       "published": true}
      """
