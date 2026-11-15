Feature: A title-only lint failure is shortened at once
  This feature shapes synthetic Intents natively with a scripted shaper and no model. A lint failure
  whose lines are all title length problems is repaired with the shortened title in the same
  attempt; any other lint line keeps today's refusal, and a title of 50 characters is untouched.

  Scenario: A 53-character title is shortened at once
    When shape-title-only-lint-fixed-in-place: the "long-title" case is shaped
    Then shape-title-only-lint-fixed-in-place: the Intent is published
    And shape-title-only-lint-fixed-in-place: no submission is refused
    And shape-title-only-lint-fixed-in-place: the stored intent.md title is "Add an is-even check for integers and also reject"
    And shape-title-only-lint-fixed-in-place: 1 title_shortened event is recorded with original "Add an is-even check for integers and also reject all" and title "Add an is-even check for integers and also reject"
    And shape-title-only-lint-fixed-in-place: progress printed the line "kogen: shortened the title to 50 characters: Add an is-even check for integers and also reject" 1 time

  Scenario: A title with another lint problem is refused with both lines
    When shape-title-only-lint-fixed-in-place: the "long-title-other" case is shaped
    Then shape-title-only-lint-fixed-in-place: the feedback to submission "c1" has a line starting "title:"
    And shape-title-only-lint-fixed-in-place: the feedback to submission "c1" has a line starting "decisions:"
    And shape-title-only-lint-fixed-in-place: the feedback to submission "c2" has a line starting "title:"
    And shape-title-only-lint-fixed-in-place: the feedback to submission "c2" has a line starting "decisions:"
    And shape-title-only-lint-fixed-in-place: the Intent is not published
    And shape-title-only-lint-fixed-in-place: the refused submissions are "1:intent 2:intent 3:intent"
    And shape-title-only-lint-fixed-in-place: 1 title_shortened event is recorded

  Scenario: A 50-character title is unchanged
    When shape-title-only-lint-fixed-in-place: the "fifty-char-title" case is shaped
    Then shape-title-only-lint-fixed-in-place: the Intent is published
    And shape-title-only-lint-fixed-in-place: no submission is refused
    And shape-title-only-lint-fixed-in-place: the stored intent.md title is "Add an is-even check for integers and also rejects"
    And shape-title-only-lint-fixed-in-place: 0 title_shortened events are recorded
    And shape-title-only-lint-fixed-in-place: progress printed the line "kogen: shortened the title to 50 characters: Add an is-even check for integers and also rejects" 0 times
