Feature: k-shape-approve-refuses-not-ready-draft: Refuse approval of a failed Shape draft
  Scenario: A refusal survives card requests, hashes and hand edits
    Then k-shape-approve-refuses-not-ready-draft: "sticky refusal" obeys the draft approval contract

  Scenario: A successful second Shape run restores normal approval
    Then k-shape-approve-refuses-not-ready-draft: "successful retry" obeys the draft approval contract

  Scenario: A hand-written never-shaped draft retains current approval behavior
    Then k-shape-approve-refuses-not-ready-draft: "never shaped" obeys the draft approval contract

  Scenario: Refusal diagnostics list the draft and name the recovery command
    Then k-shape-approve-refuses-not-ready-draft: "actionable diagnostics" obeys the draft approval contract
