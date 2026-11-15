Feature: Tracked directory evidence in either spelling
  This feature observes how Shape judges a cited directory tracked at HEAD, spelled with and without a trailing slash.

  Background:
    Given shape-tracked-directory-evidence-no-slash: a Git fixture with tracked docs/presets files, an uncommitted docs/drafts directory and the directory docs listed

  Scenario: A listed tracked directory is accepted in Decisions in both spellings
    Then shape-tracked-directory-evidence-no-slash: the observation "bare_problems" is ""
    And shape-tracked-directory-evidence-no-slash: the observation "slash_problems" is ""
    And shape-tracked-directory-evidence-no-slash: the observation "bare_context_has_guide" is "true"
    And shape-tracked-directory-evidence-no-slash: the observation "slash_context_has_guide" is "true"

  Scenario: The sourced decision ledger asks for files in both spellings
    Then shape-tracked-directory-evidence-no-slash: the observation "bare_ledger" is "- sourced decision 1: docs/presets is a directory; cite the files this shaping read"
    And shape-tracked-directory-evidence-no-slash: the observation "slash_ledger" is "- sourced decision 1: docs/presets/ is a directory; cite the files this shaping read"

  Scenario: A question citing a tracked directory is told it is a directory in both spellings
    Then shape-tracked-directory-evidence-no-slash: the observation "bare_question" is "Question 1 cites docs/presets: not shown (docs/presets is a directory; cite the files this shaping read)"
    And shape-tracked-directory-evidence-no-slash: the observation "slash_question" is "Question 1 cites docs/presets/: not shown (docs/presets/ is a directory; cite the files this shaping read)"

  Scenario: Unread files, unlisted directories and untracked directories stay refused
    Then shape-tracked-directory-evidence-no-slash: the observation "unread_file" is "- decision 1: this shaping did not read docs/README.md"
    And shape-tracked-directory-evidence-no-slash: the observation "unlisted_dir" is "- decision 1: this shaping did not list docs/presets"
    And shape-tracked-directory-evidence-no-slash: the observation "untracked_dir" is "- decision 1: docs/drafts is not a file or directory tracked at HEAD"
    And shape-tracked-directory-evidence-no-slash: the observation "untracked_question" is "Question 1 cites docs/drafts: not shown (docs/drafts is not a file tracked at HEAD)"
