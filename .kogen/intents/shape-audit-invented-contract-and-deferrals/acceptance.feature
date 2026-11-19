Feature: Shape audits invented contract details and deferred controls
  This feature observes the contract review gate through the scripted shaper.

  Background:
    Given shape-audit-invented-contract-and-deferrals: the audit harness has run

  Scenario: Invented route refused
    Then shape-audit-invented-contract-and-deferrals: the observation of "invented-route" includes:
      """
      {"refused":true,"names_kind":true,"names_scenario":true,"names_literal":true}
      """

  Scenario: Sourced literal accepted
    Then shape-audit-invented-contract-and-deferrals: the observation of "sourced-literal" includes:
      """
      {"refused":false,"exit":0}
      """

  Scenario: Deferred controls without failure modes refused
    Then shape-audit-invented-contract-and-deferrals: the observation of "one-direction" includes:
      """
      {"refused":true,"names_kind":true,"names_property":true,"names_missing_direction":true}
      """

  Scenario: Two failure modes publish
    Then shape-audit-invented-contract-and-deferrals: the observation of "both-directions" includes:
      """
      {"refused":false,"exit":0}
      """

  Scenario: Vacuous assertion refused
    Then shape-audit-invented-contract-and-deferrals: the observation of "happy-path" includes:
      """
      {"refused":true,"names_kind":true,"names_scenario":true}
      """

  Scenario: Controls that ran skip the audit
    Then shape-audit-invented-contract-and-deferrals: the observation of "controls-ran" includes:
      """
      {"refused":false,"exit":0,"audit_refusal":false,"audit_requests":0}
      """

  Scenario: Resubmission reuse
    Then shape-audit-invented-contract-and-deferrals: the observation of "resubmission" includes:
      """
      {"same_refusal":true,"extra_requests_for_unchanged":0,"changed_audited_again":true}
      """
