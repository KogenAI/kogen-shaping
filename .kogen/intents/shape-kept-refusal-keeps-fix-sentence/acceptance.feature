Feature: A cut kept-step refusal line keeps its fix sentence
  This feature checks the refusal line of a kept step as Shape records it, after the cut that bounds each line to 300 characters.

  Scenario: A long request text is shortened and the problem and fix sentence stay whole
    Given shape-kept-refusal-keeps-fix-sentence: a kept step named "Sponsors stay out of exports" whose request text is "Do not add a live sponsor to tool exports or change rankings." repeated 8 times and whose problem is "no wrong implementation names a scenario of it. Fix controls.json so a wrong implementation names this scenario."
    Then shape-kept-refusal-keeps-fix-sentence: the refusal line ends with " | problem: no wrong implementation names a scenario of it. Fix controls.json so a wrong implementation names this scenario."
    And shape-kept-refusal-keeps-fix-sentence: the refusal line starts with "- Sponsors stay out of exports (kept) | request: Do not add a live sponsor to tool exports"
    And shape-kept-refusal-keeps-fix-sentence: the request text is shortened with the cut marker before the problem
    And shape-kept-refusal-keeps-fix-sentence: the refusal line is 300 runes long

  Scenario: A short refusal line is unchanged
    Given shape-kept-refusal-keeps-fix-sentence: a kept step named "Sponsors stay out of exports" whose request text is "Do not add a live sponsor to tool exports or change rankings." repeated 1 time and whose problem is "no wrong implementation names a scenario of it. Fix controls.json so a wrong implementation names this scenario."
    Then shape-kept-refusal-keeps-fix-sentence: the refusal line is "- Sponsors stay out of exports (kept) | request: Do not add a live sponsor to tool exports or change rankings. | problem: no wrong implementation names a scenario of it. Fix controls.json so a wrong implementation names this scenario."
