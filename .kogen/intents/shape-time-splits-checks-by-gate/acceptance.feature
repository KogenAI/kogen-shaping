Feature: Shape time splits the Kogen checks by gate
  This feature covers the shape_time event and the progress line "kogen: shape time ...": they name the Kogen checks by gate, namely static checks, model checks, contract review and readiness.

  Scenario: A shape with a configured check and scripted model checks and contract review splits its Kogen checks by gate
    Given shape-time-splits-checks-by-gate: a scripted shaping run that passes a 300 ms static check, 300 ms decision and literal checks and a 150 ms contract review
    Then shape-time-splits-checks-by-gate: the shape_time event and the progress line split the Kogen checks by gate

  Scenario: A shape refused before any contract review reports no contract time
    Given shape-time-splits-checks-by-gate: a scripted shaping run refused before any contract review
    Then shape-time-splits-checks-by-gate: the shape_time event reports contract_ms 0 and the progress line names all four gates
