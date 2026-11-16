Feature: The contradiction line of an example check names the differing fields and the reason
  This feature checks the refusal line that Shape builds for an Examples row the rules contradict, and how the cut of a long refusal line treats it.

  Scenario: Contradiction line names a field the rules leave out
    When shape-example-contradiction-line-names-fields: the helper builds the contradiction line for scenario 'Log outcome', expected '{"http_status":204,"code":299}', derived '{"code":299}' and why 'the rules never state http_status'
    Then shape-example-contradiction-line-names-fields: the line is exactly 'Log outcome: differs in http_status | expected {"http_status":204,"code":299} | rules give {"code":299} | why: the rules never state http_status'

  Scenario: Contradiction line lists every differing field in alphabetical order
    When shape-example-contradiction-line-names-fields: the helper builds the contradiction line for scenario 'Retry', expected '{"state":"completed","code":503}', derived '{"state":"errored","code":503,"retries":0}' and why 'state and retries differ'
    Then shape-example-contradiction-line-names-fields: the line is exactly 'Retry: differs in retries, state | expected {"state":"completed","code":503} | rules give {"state":"errored","code":503,"retries":0} | why: state and retries differ'

  Scenario: Contradiction line for a list keeps the two values
    When shape-example-contradiction-line-names-fields: the helper builds the contradiction line for scenario 'Order', expected '["a","b"]', derived '["b","a"]' and why 'order matters'
    Then shape-example-contradiction-line-names-fields: the line is exactly 'Order: expected ["a","b"], rules give ["b","a"] | why: order matters'

  Scenario: Contradiction line puts the why on one line and omits an empty why
    When shape-example-contradiction-line-names-fields: the helper builds the contradiction line for scenario 'Wrapped', expected '{"a":1}', derived '{"a":2}' and why 'first\n  second'
    Then shape-example-contradiction-line-names-fields: the line is exactly 'Wrapped: differs in a | expected {"a":1} | rules give {"a":2} | why: first second'
    When shape-example-contradiction-line-names-fields: the helper builds the contradiction line for scenario 'Empty', expected '{"a":1}', derived '{"a":2}' and why '   \n'
    Then shape-example-contradiction-line-names-fields: the line is exactly 'Empty: differs in a | expected {"a":1} | rules give {"a":2}'

  Scenario: Cut refusal line keeps the differing field and the reason
    When shape-example-contradiction-line-names-fields: the helper builds the contradiction line for scenario 'Log HTTP outcomes without retries — code 299', expected '{"http_status":204,"posts":1,"deliveries":1,"payload_matches":true,"signed":true,"state":"completed","code":299,"error_present":false,"signature_hidden":true,"html_has_outcome":true}', derived '{"posts":1,"deliveries":1,"payload_matches":true,"signed":true,"state":"completed","code":299,"error_present":false,"signature_hidden":true,"html_has_outcome":true}' and why 'the rules do not state the http status of the replay'
    Then shape-example-contradiction-line-names-fields: the cut line is at most 300 runes
    And shape-example-contradiction-line-names-fields: the cut line starts with 'Log HTTP outcomes without retries — code 299: differs in http_status | expected {"http_status":204'
    And shape-example-contradiction-line-names-fields: the cut line contains ' ... '
    And shape-example-contradiction-line-names-fields: the cut line ends with ' | why: the rules do not state the http status of the replay'
