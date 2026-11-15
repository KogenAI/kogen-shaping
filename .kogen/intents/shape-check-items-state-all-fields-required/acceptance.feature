Feature: The literal and decision check texts state that every item carries every field
  This feature checks the model-facing instruction texts and tool schema descriptions of the literal check and the decision check, and that the schemas' required lists, enums and strictness and the secret-material sentence are unchanged.

  Scenario: The literal check instructions state that every field is required
    Given shape-check-items-state-all-fields-required: the literal and decision check instructions and tool schemas were observed
    Then shape-check-items-state-all-fields-required: the literal instructions contain "carries every field, including status"
    And shape-check-items-state-all-fields-required: the literal instructions contain "the empty string, never omitted"

  Scenario: The decision check instructions state that every field is required
    Given shape-check-items-state-all-fields-required: the literal and decision check instructions and tool schemas were observed
    Then shape-check-items-state-all-fields-required: the decision instructions contain "carries every field, including why"
    And shape-check-items-state-all-fields-required: the decision instructions contain "the empty string"
    And shape-check-items-state-all-fields-required: the decision instructions contain "never omitted"

  Scenario: The literal tool schema describes status and why as always present
    Given shape-check-items-state-all-fields-required: the literal and decision check instructions and tool schemas were observed
    Then shape-check-items-state-all-fields-required: the literal status property description contains "never omitted"
    And shape-check-items-state-all-fields-required: the literal why property description contains "never omitted"
    And shape-check-items-state-all-fields-required: the literal why property description contains "empty string"

  Scenario: The decision tool schema describes why as always present
    Given shape-check-items-state-all-fields-required: the literal and decision check instructions and tool schemas were observed
    Then shape-check-items-state-all-fields-required: the decision why property description contains "never omitted"
    And shape-check-items-state-all-fields-required: the decision why property description contains "empty string"

  Scenario: Secret items still leave the named fields empty
    Given shape-check-items-state-all-fields-required: the literal and decision check instructions and tool schemas were observed
    Then shape-check-items-state-all-fields-required: the literal instructions contain "for secret material leave literal, source_quote, package_value and why empty"

  Scenario: The schemas' required lists, enums and strictness are unchanged
    Given shape-check-items-state-all-fields-required: the literal and decision check instructions and tool schemas were observed
    Then shape-check-items-state-all-fields-required: the literal item required list is "literal,source,source_quote,package_value,why,status"
    And shape-check-items-state-all-fields-required: the decision item required list is "statement,request_quotes,verdict,intent_value,request_value,difference,why"
    And shape-check-items-state-all-fields-required: the literal schema top-level required list is "items"
    And shape-check-items-state-all-fields-required: the decision schema top-level required list is "items"
    And shape-check-items-state-all-fields-required: the literal status enum is "required,superseded,informal,secret"
    And shape-check-items-state-all-fields-required: the decision verdict enum is "follows,contradicts,consequence,unsupported"
    And shape-check-items-state-all-fields-required: the literal item schema has 6 properties, forbids additional properties and its tool is strict
    And shape-check-items-state-all-fields-required: the decision item schema has 7 properties, forbids additional properties and its tool is strict
