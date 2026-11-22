Feature: A property that no wrong implementation names says what to add
  This feature checks the refusal line of the judging of controls.json for a property that no wrong implementation names, and the lines that stay as they are.

  Scenario: A kept property with no wrong implementation says what to add
    Given shape-unnamed-property-says-fix: a kept property named "add keeps working" with request text "add" repeated 1 time is judged where there is no wrong implementation
    Then shape-unnamed-property-says-fix: the refusal lines are exactly:
      """
      - add keeps working (kept) | request: add | problem: no wrong implementation in controls.json names a scenario of it. Add a wrong implementation to controls.json that breaks this unchanged behaviour and names a scenario of it and the step line that must reject it.
      """

  Scenario: A new property that no wrong implementation names says what to add
    Given shape-unnamed-property-says-fix: a new property named "mul" with request text "mul" repeated 1 time is judged where a wrong implementation names only another property
    Then shape-unnamed-property-says-fix: the refusal lines are exactly:
      """
      - mul (new) | request: mul | problem: no wrong implementation in controls.json names a scenario of it. Add a wrong implementation to controls.json that gets this property wrong and names a scenario of it and the step line that must reject it.
      """

  Scenario: A wrong run that did not report every scenario keeps its own reason
    Given shape-unnamed-property-says-fix: a kept property named "add keeps working" with request text "add" repeated 1 time is judged where a wrong implementation names it but its run did not report every scenario
    Then shape-unnamed-property-says-fix: the refusal lines are exactly:
      """
      - add keeps working (kept) | request: add | problem: wrong implementation "w1": its run did not report every scenario as passed or failed
      """

  Scenario: A wrong implementation rejected at another step keeps its own reason
    Given shape-unnamed-property-says-fix: a kept property named "add keeps working" with request text "add" repeated 1 time is judged where a wrong implementation names it but fails at a later step
    Then shape-unnamed-property-says-fix: the refusal lines are exactly:
      """
      - add keeps working (kept) | request: add | problem: wrong implementation "w1" fails scenario "Add keeps working" at "And is-even: sub of 5 and 3 is 2", not at "Then is-even: add of 2 and 3 is 5"
      """

  Scenario: A property that a wrong implementation rejects at its step gets no line
    Given shape-unnamed-property-says-fix: a kept property named "add keeps working" with request text "add" repeated 1 time is judged where a wrong implementation names it and is rejected at its step
    Then shape-unnamed-property-says-fix: there are no refusal lines

  Scenario: A long kept request text is cut and the whole sentence stays
    Given shape-unnamed-property-says-fix: a kept property named "Sponsors stay out of exports" with request text "Do not add a live sponsor to tool exports or change rankings." repeated 8 times is judged where there is no wrong implementation
    Then shape-unnamed-property-says-fix: the cut refusal line is 300 runes long
    And shape-unnamed-property-says-fix: the cut refusal line starts with "- Sponsors stay out of exports (kept) | request: Do not add a live"
    And shape-unnamed-property-says-fix: the cut refusal line ends with " | problem: no wrong implementation in controls.json names a scenario of it. Add a wrong implementation to controls.json that breaks this unchanged behaviour and names a scenario of it and the step line that must reject it."
