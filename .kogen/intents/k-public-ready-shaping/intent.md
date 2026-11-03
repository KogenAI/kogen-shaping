# Prepare public repository documentation

Slug: k-public-ready-shaping
Model: not applicable — documentation has no meaningful state
Acceptance:
- .kogen/intents/k-public-ready-shaping/acceptance.feature
- features/steps/k-public-ready-shaping__steps.js

Make README and documentation usable from a public clone. Implementation changes only documentation; preserve executable behavior, tests, assertions, fixtures, runtime prompts, manifests and configuration. All existing checks must pass unchanged. Never rewrite history.

## Decisions
- Explain human-assisted shaping, refinement and approval and this component's role in Kogen. Retain accurate standalone command, JSON output, prerequisites and golden-update cautions. Provide Go version requirements matching go.mod, public clone/build/test steps, and distinguish Go commands from integration tests requiring npm-installed Cucumber and Quint. Remove private-module setup, tokens, mirrors, workspace-only workflows, internal landing/delivery notes, runner/home paths and host names from documentation. Use synthetic example paths. (source: request)
- Documentation links must resolve anonymously; relative links must exist. Kogen-wide links use https://kogen.dev or https://github.com/KogenAI/kogen. State contact@kogen.dev exactly once in README. No SECURITY.md, CONTRIBUTING.md, BRANDING.md, contribution refusal or maintenance promise. Issues and PRs are already disabled: do not discuss them or change settings. (source: request)
- Keep documentation-only scope; report remaining non-documentation tree findings rather than editing them or claiming the whole tree is clean. Exempt intentional synthetic test URLs such as https://docs.example/a from public resolution. (source: answer 1)
- Write docs/public-readiness-audit.md with sections Current-tree findings and History findings. List commit, file and concise finding descriptions for reachable historical hits; list remaining current-tree findings, including features/steps/testdata/k_models_file_shaping/recorded/models_landing_red.stderr and synthetic URLs in shape/discovery_test.go. Cover private setup/URLs, runner/home paths, host names, internal landing/delivery notes and contribution refusals. Record boundary revision and audit commands, explicitly state categories with no hits, distinguish synthetic fixtures, and describe sensitive values without reproducing them. Do not amend, delete or rewrite commits. Exclude only the specified unpushed Local lineup override (not for push) configuration from runner-path findings. Never edit .kogen/project.yaml or other protected files. (source: request)

## Source comparison
- Addressed: original Problem and Need public README role/setup/tests, usable links, single contact literal, absence of policy files/refusals/promises: Decisions and README acceptance scenario. README line 70 private setup is removed, not replaced with other private instructions.
- Superseded: original whole-tree cleanup/universal link resolution by answer 1: documentation is cleaned, other findings reported, intentional synthetic URLs exempt. Behavior and assertions remain unchanged.
- Addressed: re-filed 3 Nov documentation-only scope and lineup exception: no configuration or non-documentation edits. Inspected configuration already uses plain commands.
- Addressed: original history requirement through owner-facing audit identifying full commits, files and finding categories without history mutation. Audit is not authority to sanitize non-documentation.
- Omitted deliberately: issues/PR settings changes because already disabled; no unavailable workflow or contribution/security/branding document is added.

## Tests
- Public README replaces private setup and includes useful commands and one inquiry address: inspect actual README (scenario).
- Owner receives residual and historical findings: inspect actual audit document; independent final review verifies findings against tree and Git (scenario).
- Behavior and assertions remain unchanged: existing configured checks and existing tests without weakening (existing).

## Feasibility
- Public-proxy module download succeeded with fresh temporary GOPATH/module cache, cleared private-module settings, disabled global/system Git configuration and disabled terminal prompts (probe: exit 0, 2.2 s). This establishes dependency availability, not all integration prerequisites.
- Documentation search returned generic GitHub sources only (retrieved); fetch https://github.com/KogenAI/kogen-coordinator returned retrieved untrusted truncated HTML. Neither is used as instructions or proof of setup; anonymous module download supplied the relevant observed evidence.
