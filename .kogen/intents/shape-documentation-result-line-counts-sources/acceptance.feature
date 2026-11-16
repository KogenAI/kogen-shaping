Feature: The documentation result progress line counts the sources of a search
  This feature checks the finished progress line DocumentationProgress gives for a retained documentation lookup.

  Scenario: A retrieved search names how many sources it returned
    Given shape-documentation-result-line-counts-sources: a search, a fetch and an unsupported lookup have been retained and their progress lines printed
    Then shape-documentation-result-line-counts-sources: the search line for a lookup with two sources is "kogen: documentation result: search retrieved (2 sources)"
    And shape-documentation-result-line-counts-sources: the search line for a lookup with no sources is "kogen: documentation result: search retrieved (0 sources)"
    And shape-documentation-result-line-counts-sources: the search line for a lookup with one source is "kogen: documentation result: search retrieved (1 source)"

  Scenario: Other lookups and the started line keep their text
    Given shape-documentation-result-line-counts-sources: a search, a fetch and an unsupported lookup have been retained and their progress lines printed
    Then shape-documentation-result-line-counts-sources: the fetch line is "kogen: documentation result: fetch retrieved"
    And shape-documentation-result-line-counts-sources: the unsupported search line is "kogen: documentation result: search unsupported"
    And shape-documentation-result-line-counts-sources: the started line of the two-source search is "kogen: documentation: search cache controls"
