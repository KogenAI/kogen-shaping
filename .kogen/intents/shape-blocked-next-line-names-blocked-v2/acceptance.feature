Feature: Name only the blocked files in the Next advice
  This feature reproduces pattern blocked-next-line-names-wrong-file from ev-713f0ef2.
  Since 19 Nov four attempts across four rows (add-kogen-and-optimum-credo,
  intent-recheck-retained-draft, role-magic-link-onboarding) were refused with
  shape/protected_conflict. Reported protected paths included mix.exs, cli/go.mod,
  cli/go.sum and .kogen/project.yaml, but fixed advice sent every report to the
  project definition even when it was not listed. Only the third output line changes.

  Scenario: Report a protected settings file
    When shape-blocked-next-line-names-blocked-v2: the blocked report for "glob" runs
    Then shape-blocked-next-line-names-blocked-v2: the report is exactly these three lines and exits 1 without an Intent or approval
      """
      shape/protected_conflict: required edits touch protected files: settings/check.cfg
      the requested verification definition cannot be changed in Build
      Next: revise the request so it needs no edit to the files listed above, or review and commit those files outside Build, then shape again; preserve checks and actual check configuration. There is no protected-edit override.
      """

  Scenario: Report the protected project definition
    When shape-blocked-next-line-names-blocked-v2: the blocked report for "project" runs
    Then shape-blocked-next-line-names-blocked-v2: the report is exactly these three lines and exits 1 without an Intent or approval
      """
      shape/protected_conflict: required edits touch protected files: .kogen/project.yaml
      the requested check definition must change
      Next: revise the request so it needs no edit to the files listed above, or review and commit those files outside Build, then shape again; preserve checks and actual check configuration. There is no protected-edit override.
      """

  Scenario: Report two dependency manifests
    When shape-blocked-next-line-names-blocked-v2: the blocked report for "manifest" runs
    Then shape-blocked-next-line-names-blocked-v2: the report is exactly these three lines and exits 1 without an Intent or approval
      """
      shape/protected_conflict: required edits touch protected files: cli/go.mod, cli/go.sum
      the module dependencies must change
      Next: revise the request so it needs no edit to the files listed above, or review and commit those files outside Build, then shape again; preserve checks and actual check configuration. There is no protected-edit override.
      """

  Scenario Outline: Run the <cardinality> repair journey
    When shape-blocked-next-line-names-blocked-v2: the repair journey in "<mode>" mode runs
    Then shape-blocked-next-line-names-blocked-v2: the repair journey passes with this Next line
      """
      Next: revise the request so it needs no edit to the files listed above, or review and commit those files outside Build, then shape again; preserve checks and actual check configuration. There is no protected-edit override.
      """
    Examples:
      | cardinality | mode      |
      | single-path | mixed     |
      | two-path    | protected |

  Scenario: Update the existing Go output expectations
    When shape-blocked-next-line-names-blocked-v2: the affected existing Go tests run
    Then shape-blocked-next-line-names-blocked-v2: both existing Go expectations pass
