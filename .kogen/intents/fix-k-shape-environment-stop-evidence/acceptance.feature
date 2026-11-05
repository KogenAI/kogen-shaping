Feature: Cited evidence never follows a tracked symlink
  For the external shaper, a question's (evidence: <path>) item that names a file tracked at HEAD
  is shown to the question check with the file's checkout text. A cited path that is a symbolic
  link in the checkout is not read: its block says it is not shown because it is a symbolic link,
  so the text of a file it points to, inside or outside the checkout, never reaches the question
  check. A regular tracked file cited beside it is still shown as before.

  Scenario: A tracked symlink to a file outside the checkout is not shown to the question check
    When fix-k-shape-environment-stop-evidence: the external shaper asks a question citing a tracked file and a tracked symlink to a file outside the checkout
    Then fix-k-shape-environment-stop-evidence: the observations include:
      """
      {
        "exit": 5,
        "error": "",
        "sessions": 2,
        "tree_entry_mode": "120000",
        "secret_sent": false,
        "cited_evidence": "Question 1 cites lib/weather.js (read from the checkout):\n1\t// The weather client. No fake client exists: every call reaches the live service.\n2\texports.client = function () {}\n\nQuestion 1 cites lib/secret.js: not shown (lib/secret.js is a symbolic link)"
      }
      """
