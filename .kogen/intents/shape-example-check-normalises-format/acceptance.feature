Feature: Normalise shape example comparisons
  Scenario Outline: Classify example values by meaning
    When shape-example-check-normalises-format: Shape checks expected <expected> against derived <derived>
    Then shape-example-check-normalises-format: the verdict is <verdict> and example refusal is <refused>
    Examples:
      | expected                       | derived                              | verdict     | refused |
      | "status":0,"out":"61610a0a" | {"status":0,"out":"61610a0a"} | follows     | false   |
      | "status":0,"out":"61610a0a" | status:0, out:61610a0a                | follows     | false   |
      | "status":0,"out":"61610a0a" | {"status":1,"out":"61610a0a"} | contradicts | true    |
      | "status":0,"out":"61610a0a" | <not-given>                          | unsupported | false   |
