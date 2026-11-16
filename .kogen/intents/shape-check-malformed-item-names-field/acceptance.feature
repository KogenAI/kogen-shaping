Feature: Name malformed checker items and fields
  This feature reproduces model-free the omitted status and why payloads reported
  in AoR climb release 891c91b3, h14 to h17: 148 of 414 items submissions rejected,
  comprising 123 literal submissions omitting status and 25 decision submissions
  omitting why. The lab example h17-replicate-4-vs-baseline-891c91b3/cells/rails-ft-saas-billing__baseline__r3
  had StructuredOutput result_error true in executor-literal-1 and executor-literal-2
  in shaping-events.ndjson, followed by accepted calls with status present. These
  are provenance, not dependencies on historical logs, Rails, Studio, or a model.

  Scenario: Diagnose malformed literal items without changing validity
    Given shape-check-malformed-item-names-field: the literal parser cases
      """
      [
        {"name":"missing_status_at_item_3","items":[{"literal":"l","source":"request","source_quote":"q","package_value":"","why":"w","status":"required"},{"literal":"l","source":"request","source_quote":"q","package_value":"","why":"w","status":"required"},{"literal":"l","source":"request","source_quote":"q","package_value":"","why":"w"}],"want":"item 3 is malformed: status is missing"},
        {"name":"wrong_status_at_item_2","items":[{"literal":"l","source":"request","source_quote":"q","package_value":"","why":"w","status":"required"},{"literal":"l","source":"request","source_quote":"q","package_value":"","why":"w","status":"maybe"}],"want":"item 2 is malformed: status must be one of required, superseded, informal, secret"},
        {"name":"unknown_verdict_at_item_5","items":[{"literal":"l","source":"request","source_quote":"q","package_value":"","why":"w","status":"required"},{"literal":"l","source":"request","source_quote":"q","package_value":"","why":"w","status":"required"},{"literal":"l","source":"request","source_quote":"q","package_value":"","why":"w","status":"required"},{"literal":"l","source":"request","source_quote":"q","package_value":"","why":"w","status":"required"},{"literal":"l","source":"request","source_quote":"q","package_value":"","why":"w","status":"required","verdict":"follows"}],"want":"item 5 is malformed: unknown field verdict"},
        {"name":"all_three_omit_status","items":[{"literal":"l","source":"request","source_quote":"q","package_value":"","why":"w"},{"literal":"l","source":"request","source_quote":"q","package_value":"","why":"w"},{"literal":"l","source":"request","source_quote":"q","package_value":"","why":"w"}],"want":"item 1 is malformed: status is missing (3 malformed items)"},
        {"name":"first_item_not_first_problem","items":[{"literal":"l","source":"request","source_quote":"q","package_value":"","why":"w","status":"required"},{"literal":"l","source":"request","source_quote":"q","package_value":""},{"literal":"l","source":"request","source_quote":"q","package_value":"","why":"w","status":"maybe"}],"want":"item 2 is malformed: why is missing (2 malformed items)"},
        {"name":"wrong_field_type","items":[{"literal":"l","source":"request","source_quote":"q","package_value":"","why":"w","status":"required"},{"literal":7,"source":"request","source_quote":"q","package_value":"","why":"w","status":"required"}],"want":"item 2 is malformed: literal must be a string"},
        {"name":"wrong_status_type","items":[{"literal":"l","source":"request","source_quote":"q","package_value":"","why":"w","status":null}],"want":"item 1 is malformed: status must be a string"},
        {"name":"nonobject","items":[{"literal":"l","source":"request","source_quote":"q","package_value":"","why":"w","status":"required"},"x"],"want":"item 2 is malformed: must be an object"},
        {"name":"unknown_fields_are_sorted","items":[{"literal":"l","source":"request","source_quote":"q","package_value":"","why":"w","status":"required","zeta":"","alpha":""}],"want":"item 1 is malformed: unknown field alpha"},
        {"name":"all_allowed_statuses","items":[{"literal":"l","source":"request","source_quote":"q","package_value":"","why":"w","status":"required"},{"literal":"l","source":"request","source_quote":"q","package_value":"","why":"w","status":"superseded"},{"literal":"l","source":"request","source_quote":"q","package_value":"","why":"w","status":"informal"},{"literal":"l","source":"request","source_quote":"q","package_value":"","why":"w","status":"secret"}],"want":""},
        {"name":"empty_strings","items":[{"literal":"","source":"","source_quote":"","package_value":"","why":"","status":"required"}],"want":""},
        {"name":"empty_items","items":[],"want":""}
      ]
      """
    Then shape-check-malformed-item-names-field: named test "TestMalformedLiteralItemDiagnostics" passes every supplied case

  Scenario: Diagnose malformed decision items without changing validity
    Given shape-check-malformed-item-names-field: the decision parser cases
      """
      [
        {"name":"missing_why_at_item_4","items":[{"statement":"s","request_quotes":["q"],"verdict":"follows","intent_value":"","request_value":"","difference":"","why":"w"},{"statement":"s","request_quotes":["q"],"verdict":"follows","intent_value":"","request_value":"","difference":"","why":"w"},{"statement":"s","request_quotes":["q"],"verdict":"follows","intent_value":"","request_value":"","difference":"","why":"w"},{"statement":"s","request_quotes":["q"],"verdict":"follows","intent_value":"","request_value":"","difference":""}],"want":"item 4 is malformed: why is missing"},
        {"name":"wrong_verdict_at_item_2","items":[{"statement":"s","request_quotes":["q"],"verdict":"follows","intent_value":"","request_value":"","difference":"","why":"w"},{"statement":"s","request_quotes":["q"],"verdict":"maybe","intent_value":"","request_value":"","difference":"","why":"w"}],"want":"item 2 is malformed: verdict must be one of follows, contradicts, consequence, unsupported"},
        {"name":"unknown_status","items":[{"statement":"s","request_quotes":["q"],"verdict":"follows","intent_value":"","request_value":"","difference":"","why":"w","status":"required"}],"want":"item 1 is malformed: unknown field status"},
        {"name":"two_omit_why","items":[{"statement":"s","request_quotes":["q"],"verdict":"follows","intent_value":"","request_value":"","difference":"","why":"w"},{"statement":"s","request_quotes":["q"],"verdict":"follows","intent_value":"","request_value":"","difference":""},{"statement":"s","request_quotes":["q"],"verdict":"follows","intent_value":"","request_value":"","difference":""}],"want":"item 2 is malformed: why is missing (2 malformed items)"},
        {"name":"first_problem_in_field_order","items":[{"statement":"s","request_quotes":["q"],"verdict":"follows","intent_value":"","request_value":"","difference":"","why":"w"},{"request_quotes":["q"],"verdict":"follows","intent_value":"","request_value":"","difference":""},{"statement":"s","request_quotes":["q"],"verdict":"maybe","intent_value":"","request_value":"","difference":"","why":"w"}],"want":"item 2 is malformed: statement is missing (2 malformed items)"},
        {"name":"wrong_field_type","items":[{"statement":"s","request_quotes":["q"],"verdict":"follows","intent_value":"","request_value":"","difference":"","why":"w"},{"statement":"s","request_quotes":["q"],"verdict":"follows","intent_value":"","request_value":"","difference":"","why":3}],"want":"item 2 is malformed: why must be a string"},
        {"name":"wrong_quotes_type","items":[{"statement":"s","request_quotes":"q","verdict":"follows","intent_value":"","request_value":"","difference":"","why":"w"}],"want":"item 1 is malformed: request_quotes must be a list"},
        {"name":"wrong_quote_element","items":[{"statement":"s","request_quotes":["q"],"verdict":"follows","intent_value":"","request_value":"","difference":"","why":"w"},{"statement":"s","request_quotes":["q",3],"verdict":"follows","intent_value":"","request_value":"","difference":"","why":"w"}],"want":"item 2 is malformed: request_quotes must contain only strings"},
        {"name":"nonobject","items":[{"statement":"s","request_quotes":["q"],"verdict":"follows","intent_value":"","request_value":"","difference":"","why":"w"},null],"want":"item 2 is malformed: must be an object"},
        {"name":"unknown_fields_are_sorted","items":[{"statement":"s","request_quotes":["q"],"verdict":"follows","intent_value":"","request_value":"","difference":"","why":"w","zeta":"","alpha":""}],"want":"item 1 is malformed: unknown field alpha"},
        {"name":"all_allowed_verdicts","items":[{"statement":"s","request_quotes":["q"],"verdict":"follows","intent_value":"","request_value":"","difference":"","why":"w"},{"statement":"s","request_quotes":["q"],"verdict":"contradicts","intent_value":"","request_value":"","difference":"","why":"w"},{"statement":"s","request_quotes":["q"],"verdict":"consequence","intent_value":"","request_value":"","difference":"","why":"w"},{"statement":"s","request_quotes":["q"],"verdict":"unsupported","intent_value":"","request_value":"","difference":"","why":"w"}],"want":""},
        {"name":"empty_strings_and_quotes","items":[{"statement":"","request_quotes":[],"verdict":"follows","intent_value":"","request_value":"","difference":"","why":""}],"want":""},
        {"name":"empty_items","items":[],"want":""}
      ]
      """
    Then shape-check-malformed-item-names-field: named test "TestMalformedDecisionItemDiagnostics" passes every supplied case

