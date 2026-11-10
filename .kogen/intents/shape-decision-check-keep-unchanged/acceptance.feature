Feature: Support evidenced keep-unchanged decisions
  This feature pins the existing keep-unchanged policy and exercises its local
  decision_check gate with scripted model results. It does not claim a live model
  evaluation. The policy already exists; file-and-line citations still need correction.
  Historical motivation: on 8 Nov, 6 of 9 intent/not_ready failures were Shape
  gates: decision_check 2, contract_review 2, carried_decisions 1, decision_evidence 1.
  Studio issue ed7490a5-37ca-4fc0-b7c6-30664f7efcbd requested this correction.
  The retry basis is v0.1.0-dev.3c70abcb, including shape-unreachable-names-build-error
  at 4c0d87f; compiler/step-error forwarding is not changed here.

  Scenario: Keep existing controls values without request quotes
    Given shape-decision-check-keep-unchanged: the local checker case
      """
      {"Statement":"controlsMaxRuns (8), controlsMaxFiles (8), controlsMaxBytes (65536) and controlsTimeCap keep their values.","Citation":"limits.go:2","Evidence":"const controlsMaxFiles = 8","Verdict":"follows","Policy":["For a Decision that preserves an existing repository value, use the observed repository evidence below: if it shows the stated current value, give follows without requiring a request quote."]}
      """
    When shape-decision-check-keep-unchanged: decision_check evaluates the candidate
    Then shape-decision-check-keep-unchanged: the evidence-backed verdict and checker policy hold

  Scenario: Keep unchanged helper behaviour supported by a scratch observation
    Given shape-decision-check-keep-unchanged: the local checker case
      """
      {"Statement":"Keep helpers in features/steps.","Citation":"ran `observe helpers`","Evidence":"helpers directory: features/steps","Probe":true,"Verdict":"follows","Policy":["if it shows the stated current value, give follows without requiring a request quote."]}
      """
    When shape-decision-check-keep-unchanged: decision_check evaluates the candidate
    Then shape-decision-check-keep-unchanged: the evidence-backed verdict and checker policy hold

  Scenario: Refuse a new choice without quote evidence or assumption
    Given shape-decision-check-keep-unchanged: the local checker case
      """
      {"Statement":"Increase controlsMaxRuns to 9.","Verdict":"unsupported","Policy":["A new choice with neither evidence nor a recorded assumption remains unsupported."]}
      """
    When shape-decision-check-keep-unchanged: decision_check evaluates the candidate
    Then shape-decision-check-keep-unchanged: the evidence-backed verdict and checker policy hold

  Scenario: Refuse a keep-unchanged claim that does not match cited evidence
    Given shape-decision-check-keep-unchanged: the local checker case
      """
      {"Statement":"Keep controlsMaxFiles at 9.","Citation":"limits.go:3","Evidence":"const controlsMaxFiles = 8","Verdict":"unsupported","Policy":["A citation name alone is not proof.","Give unsupported when the observed evidence does not establish the claimed value, including a keep-unchanged claim with a different value.","Explicit request conflicts still contradict even when the repository currently differs."]}
      """
    When shape-decision-check-keep-unchanged: decision_check evaluates the candidate
    Then shape-decision-check-keep-unchanged: the evidence-backed verdict and checker policy hold

