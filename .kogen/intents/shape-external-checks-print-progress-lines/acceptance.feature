Feature: Announce external decision and literal checks
  This feature reproduces the silent external check branches using local scripted
  executors, not models, browsers, Rails or Studio tooling. Historical context:
  AoR climb release 891c91b3, cell
  h19-chatgpt-sol-terra-vs-baseline-891c91b3/cells/rails-ft-data-export-and-import__baseline__r2,
  stderr.txt and shaping-events.ndjson reported shape time 978.8 s and checks 693.7 s.
  Readiness at 272 s followed by external round 2 at 450 s concealed 178 s of
  literal_check and decision_check work; later gaps were 21 s, including readiness
  at 577 s and round 3 at 598 s, in rounds 2 to 5. The native comparison was
  rails-ft-saas-billing__chatgpt-sol-terra. These are supplied observations, not a
  requirement to reproduce historical wall times or access the external lab.
  Native response shorthand remains decision check response ... wall_ms= and
  literal check response ... wall_ms=; ellipses denote the existing usage fields.
  Lab evidence: /home/lanes/work.noindex/aor-climb/hypotheses/h19-chatgpt-sol-terra-vs-baseline-891c91b3/cells/rails-ft-data-export-and-import__baseline__r2/run/cells/rails-ft-data-export-and-import/r1/baseline/stderr.txt
  Lines 8-9, 14-15, 20-21, 26-27: 178, 21, 21, 21 s gaps; checks_ms 693723.

  Background:
    Given shape-external-checks-print-progress-lines: the scripted check expectations
      """
      {"rounds":2,"decision":{"start":"kogen: checking the stated decisions","finish":"kogen: decision check finished","response":"kogen: decision check response"},"literal":{"start":"kogen: checking the required literals","finish":"kogen: literal check finished","response":"kogen: literal check response"}}
      """

  Scenario: External checks announce every completed round
    When shape-external-checks-print-progress-lines: an external Shape completes two checked submissions
    Then shape-external-checks-print-progress-lines: each external check announces its round before and after execution
    When shape-external-checks-print-progress-lines: a native Shape completes a checked submission
    Then shape-external-checks-print-progress-lines: native check progress retains its existing format
