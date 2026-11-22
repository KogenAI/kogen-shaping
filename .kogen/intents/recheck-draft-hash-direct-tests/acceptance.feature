Feature: RecheckDraftHash has direct regression tests
  This feature checks the new recheck test file of package shape. Its tests pass against the real
  RecheckDraftHash, and each of five broken copies of that function makes at least one of them fail.

  Scenario: The tests of the recheck test file pass against the real function
    When recheck-draft-hash-direct-tests: I run the tests declared in the recheck test file
    Then recheck-draft-hash-direct-tests: the go command exits 0
    And recheck-draft-hash-direct-tests: every declared test is reported as passed
    And recheck-draft-hash-direct-tests: no Go file other than the recheck test file is changed

  Scenario: The recheck tests reject each broken hash function
    When recheck-draft-hash-direct-tests: I run those tests against copies of RecheckDraftHash broken in five ways
    Then recheck-draft-hash-direct-tests: the recheck tests fail when the hash is empty
    And recheck-draft-hash-direct-tests: the recheck tests fail when the hash is the same for every draft
    And recheck-draft-hash-direct-tests: the recheck tests fail when the hash differs between two calls
    And recheck-draft-hash-direct-tests: the recheck tests fail when the hash ignores acceptance.feature
    And recheck-draft-hash-direct-tests: the recheck tests fail when the project configuration error is swallowed
