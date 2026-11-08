Feature: Explain Quint invariant expressions
  Scenario: Shaping prose explains multiple invariants
    Then k-shape-quint-invariant-expression: "shape/prompts/shaper.md" contains the guidance
      """
      For multiple invariants, pass one Boolean conjunction expression, e.g. `quint run model.qnt --backend=typescript --invariant='A and B and C'`; a comma list such as `--invariant=A,B,C` is not valid. Commas belong only to Intent metadata `invariants=A,B,C`.
      """

  Scenario: Model-facing snapshot matches the guidance
    Then k-shape-quint-invariant-expression: the policy prompt snapshot contains the guidance
      """
      For multiple invariants, pass one Boolean conjunction expression, e.g. `quint run model.qnt --backend=typescript --invariant='A and B and C'`; a comma list such as `--invariant=A,B,C` is not valid. Commas belong only to Intent metadata `invariants=A,B,C`.
      """
