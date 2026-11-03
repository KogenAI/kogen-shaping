Feature: Shaping stops reading the removed opt-ins and fixes their defaults
  The coordinator removes shape_workers, shape_audit, shape_decision_check, review_material and
  every role's max_rounds, and its candidates refuse a project.yaml that sets any of them. Shaping
  reads none of them and always behaves as with today's defaults: no shape workers or batch tool,
  no request audit, no decision check, and the executor's own round cap for a claude-code shaper.
  The coordinator that rejects the keys is synthesized from the pinned one in a temporary folder;
  the behaviour scenarios run against the pinned coordinator, where a project can still set them.

  Scenario: kogen-shaping builds, vets and passes its tests against a coordinator that rejects the removed keys
    Given kso: the pinned coordinator, synthesized to reject the shape_workers, shape_audit, shape_decision_check, review_material and max_rounds keys and to hide their fields
    Then kso: go build, go vet and go test of every kogen-shaping package pass against it

  Scenario: No shaping test or fixture writes a removed key
    Then kso: no file under shape or cmd names shape_workers, shape_audit, shape_decision_check, review_material or max_rounds

  Scenario: A native Shape of a project that still sets every opt-in behaves as with today's defaults
    Then kso: a native Shape of a project that sets every opt-in offers only the default tools, answers shape_batch as an unknown tool, makes no audit or decision check request and publishes the submitted Intent unchanged

  Scenario Outline: A claude-code shaper's round cap is the executor's own whatever max_rounds says, and 6 when the executor has none
    Then kso: a claude-code shaper whose role sets max_rounds 1 under an executor shape cap of <cap> rounds gets a cap of <want> rounds and publishes after 2 rounds

    Examples:
      | cap | want |
      | 3   | 3    |
      | 0   | 6    |

  Scenario: A claude-code shaper of a project that still sets every opt-in is not refused and runs no audit or decision check session
    Then kso: a claude-code shaper of a project that sets every opt-in is not refused and publishes the submitted Intent from one session of one round

  Scenario: The policy goldens drop only the worker entries
    Then kso: the policy goldens drop only the shape_batch, worker_result and worker-role entries and keep every other entry unchanged
