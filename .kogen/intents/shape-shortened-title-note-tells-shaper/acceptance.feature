Feature: Tell the shaper when its title was shortened
  This feature covers the Shape submission result. When Shape shortens a title longer than 50 characters,
  the text the shaper receives for that submission starts with a note line naming the shortened title,
  so the shaper keeps it. The long title used is "Add an is-even check for integers and also reject all"
  (53 characters). The progress line, the title_shortened event and every gate result are unchanged.

  Scenario: A shortened title is announced to the shaper in the submission output
    Given shape-shortened-title-note-tells-shaper: the supplied scripted Shape observer is available
    When shape-shortened-title-note-tells-shaper: the scripted shapings run
    Then shape-shortened-title-note-tells-shaper: the "refused" output for c1 starts with
      """
      note:
      the title was shortened to 50 characters: Add an is-even check for integers and also reject; keep this title in later submissions
      """

  Scenario: A title-only lint failure is shortened once and the shaping succeeds
    Given shape-shortened-title-note-tells-shaper: the supplied scripted Shape observer is available
    When shape-shortened-title-note-tells-shaper: the scripted shapings run
    Then shape-shortened-title-note-tells-shaper: the "lint-only" case recorded 1 title_shortened event(s), 1 progress line(s) and exit 0
    And shape-shortened-title-note-tells-shaper: the "lint-only" title_shortened event has original "Add an is-even check for integers and also reject all" and title "Add an is-even check for integers and also reject"

  Scenario: A fifty-character title gets no shortened-title note
    Given shape-shortened-title-note-tells-shaper: the supplied scripted Shape observer is available
    When shape-shortened-title-note-tells-shaper: the scripted shapings run
    Then shape-shortened-title-note-tells-shaper: the "fifty" output for c1 is not empty and lacks "the title was shortened"
    And shape-shortened-title-note-tells-shaper: the "fifty" case recorded 0 title_shortened event(s), 0 progress line(s) and exit 0
