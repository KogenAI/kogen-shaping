Feature: Exclude managed runtime installation from Shape time charged to the budget
  This feature preserves the request's cold-host reproducer: AoR climb release 30cf6619,
  cell h6-release-30cf6619-vs-4fb52c19/cells/rails-ft-data-export-and-import__release-30cf6619
  was refused after stderr.txt reported installing Claude Code 2.1.318, installed Claude
  Code 2.1.318 with sha256, then shaping external round 1 at 115 s. Its
  shaping-events.ndjson shape_time had elapsed_ms 908023, shaper_ms 280916,
  checks_ms 511168 and unexplained gap_ms 115939. The other 50 shaped cells had
  0.6-3.1 s gaps; rails-ft-saas-billing__release-30cf6619 also installed the runtime.
  These are historical observations, not dependencies on those external artifacts.
  The executable replay below replaces downloading with a cold runtime preparation
  and retains the original millisecond arithmetic: charged time becomes 792084 ms.
  No new configuration is introduced. The model, tools and checks retain the
  existing 15-minute budget; install time alone is excluded. Refusal texts stay
  unchanged except positive install_ms on the budget gate and timing lines.

  Scenario: Replay the reported cold-host refusal without charging installation
    When shape-time-excludes-runtime-install: Shape runs this scripted timing:
      | install_ms | 115939 |
      | shaper_ms | 280916 |
      | checks_ms | 511168 |
      | startup_ms | 0 |
      | warm | false |
      | mode | ready |
    Then shape-time-excludes-runtime-install: Shape exits 0 after 2 main rounds
    And shape-time-excludes-runtime-install: the account is:
      | elapsed_ms | 908023 |
      | shaper_ms | 280916 |
      | checks_ms | 511168 |
      | install_ms | 115939 |
      | gap_ms | 0 |
    And shape-time-excludes-runtime-install: the progress line is:
      """
      kogen: shape time 908.0 s: shaper 280.9 s (model unknown, tools unknown), Kogen checks 511.2 s (static 0.0 s, model checks 0.0 s, contract review 511.2 s, readiness 0.0 s), gaps 0.0 s, runtime install 115.9 s
      """

  Scenario: A warm runtime retains the old progress line and counts ordinary startup
    When shape-time-excludes-runtime-install: Shape runs this scripted timing:
      | install_ms | 0 |
      | shaper_ms | 280916 |
      | checks_ms | 511168 |
      | startup_ms | 2000 |
      | warm | true |
      | mode | ready |
    Then shape-time-excludes-runtime-install: Shape exits 0 after 2 main rounds
    And shape-time-excludes-runtime-install: the progress line is:
      """
      kogen: shape time 794.1 s: shaper 280.9 s (model unknown, tools unknown), Kogen checks 511.2 s (static 0.0 s, model checks 0.0 s, contract review 511.2 s, readiness 0.0 s), gaps 2.0 s
      """
    And shape-time-excludes-runtime-install: the account is:
      | elapsed_ms | 794084 |
      | shaper_ms | 280916 |
      | checks_ms | 511168 |
      | install_ms | 0 |
      | gap_ms | 2000 |

  Scenario Outline: The budget still refuses at 900 charged seconds and names positive install time
    When shape-time-excludes-runtime-install: Shape runs this scripted timing:
      | install_ms | <install> |
      | shaper_ms | 900000 |
      | checks_ms | 0 |
      | startup_ms | 0 |
      | warm | <warm> |
      | mode | refuse |
    Then shape-time-excludes-runtime-install: Shape exits 1 after 1 main rounds
    And shape-time-excludes-runtime-install: the budget refusal preserves its wording and reports install_ms <install>
    And shape-time-excludes-runtime-install: the account is:
      | elapsed_ms | <elapsed> |
      | shaper_ms | 900000 |
      | checks_ms | 0 |
      | install_ms | <install> |
      | gap_ms | 0 |
    Examples:
      | install | warm | elapsed |
      | 115939 | false | 1015939 |
      | 0 | true | 900000 |

  Scenario: Time just inside the unchanged budget permits the next round despite a cold install
    When shape-time-excludes-runtime-install: Shape runs this scripted timing:
      | install_ms | 115939 |
      | shaper_ms | 899999 |
      | checks_ms | 0 |
      | startup_ms | 0 |
      | warm | false |
      | mode | ready |
    Then shape-time-excludes-runtime-install: Shape exits 0 after 2 main rounds
    And shape-time-excludes-runtime-install: the account is:
      | elapsed_ms | 1015938 |
      | shaper_ms | 899999 |
      | checks_ms | 0 |
      | install_ms | 115939 |
      | gap_ms | 0 |
