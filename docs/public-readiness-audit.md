# Public readiness audit

Boundary: `13c9b212d3f36ecacd180729d0af7ab2a753db55` (the current local
boundary, including the unpushed Local lineup override). This is an inventory, not
a declaration that the whole tree is clean. Sensitive path values are described,
not reproduced. This documentation-only change does not amend, delete or rewrite
history or change executable material. The override's configuration alone is
excluded from runner-path findings, as requested.

## Current-tree findings

- `features/steps/testdata/k_models_file_shaping/recorded/models_landing_red.stderr`:
  recorded Go panic stack contains a runner's home and tool installation paths.
- `shape/discovery_test.go`: synthetic example-domain URLs and loopback addresses
  are test inputs, not links intended for public resolution. Other synthetic
  network references occur in `shape/lineup_shape_test.go`,
  `shape/unpassable_test.go`, and `features/steps/k-shape-cmd-entry__steps.js`.
- The approved historical Intents under `.kogen/intents/` retain internal
  landing and delivery context. In particular, `k-golden-update-shaping`,
  `k-models-file-shaping`, `k-shape-cmd-entry`,
  `k-shape-decide-non-ux-general`, `k-shape-drop-optins`,
  `k-shape-prompt-move`, `k-shape-runner-guidance-general`,
  `k-shape-selfcheck-stderr`, `k-step-file-collision`, and
  `k-verify-gaps-shaping` contain such notes in their `intent.md` files.
  These protected specifications are not public setup instructions.
- The README's former private-module setup has been removed. Outside the
  excluded override and the recorded stack above, the searches found no other
  runner/home paths. No private setup URLs, tokens, mirrors, actual host names,
  or contribution refusals were identified in the current tracked tree.
  Module import paths and public license URLs are not private setup URLs.

## History findings

The following identifies the first reachable commit introducing each finding;
its unchanged content can persist in descendant trees. All commits reachable
from the boundary were searched, not just the listed introductions.

- `94eabe5d492dadf27918132feb2b1849b359f712` — `README.md`:
  private-module setup instruction, retained through the boundary.
- `37d547497cc994e870f49138affa55ff7dfd4060` —
  `.kogen/intents/k-shape-decide-non-ux/intent.md`: private research location,
  personal-machine reference and internal landing notes. Removed from the tree
  by `993411d9de5f2938ce48b1a363a823691029435d`, but still reachable.
- `e572010734c34b621773cd7f401bd08218931bac` —
  `features/steps/testdata/k_models_file_shaping/recorded/models_landing_red.stderr`:
  recorded runner home/tool paths, still present.
- Internal landing/delivery notes also appear in the Intent files listed above;
  the following table records the reachable introducing commits and files.

- `f561b3df6f00507ebc7249857f0662736a1769d0` — `.kogen/intents/k-golden-update-shaping/intent.md`: internal landing/delivery context.
- `e572010734c34b621773cd7f401bd08218931bac` — `.kogen/intents/k-models-file-shaping/intent.md`: internal landing/delivery context.
- `acd50c60e05ab0a8b6994c2608539f0bde01f2cf` — `.kogen/intents/k-shape-cmd-entry/intent.md`: internal landing/delivery context.
- `993411d9de5f2938ce48b1a363a823691029435d` — `.kogen/intents/k-shape-decide-non-ux-general/intent.md`: internal landing/delivery context.
- `f04edcea9896db525855a810bf6c0429b8494278` — `.kogen/intents/k-shape-drop-optins/intent.md`: internal landing/delivery context.
- `9a4835e496054c37e265c7300582a46ead896bee` — `.kogen/intents/k-shape-prompt-move/intent.md`: internal landing/delivery context.
- `8f0ffb9415ab4575de97c7016f82026869358bfe` — `.kogen/intents/k-shape-runner-guidance-general/intent.md`: internal landing/delivery context.
- `0f3c76f5c6db3e6bf85ab2f570f11fe53ab31bfa` — `.kogen/intents/k-shape-selfcheck-stderr/intent.md`: internal landing/delivery context.
- `bad4db349be887c328ce1200c9d053558edc78c7` — `.kogen/intents/k-verify-gaps-shaping/intent.md`: internal landing/delivery context.

- `3e17221043f4d0e804e6d843b1e1be11ed25d2d6` —
  `.kogen/intents/k-step-file-collision/intent.md`: internal delivery evidence.
- `13ff1327fab58361289940f28e30728d2526e3d6` —
  `.kogen/intents/k-shape-data-stores-no-tool/intent.md`: historical landing context.

No historical private setup URLs, tokens, mirrors, actual host names or
contribution refusals were identified by these searches. Historical synthetic
URLs are test data, not public links. A personal-machine reference in the old
Intent is not treated as a network host name. Search results should be assessed
in context; this is not a secret-scanner guarantee.

### Method

From the repository root, the audit used `git rev-parse HEAD`,
`git log --all --format='%H'`, and `git log --all -p -- README.md`.
For each reachable commit, `git grep -inE` searched its tree for private setup
terms (`GOPRIVATE`, `insteadOf`, private modules), runner/home path patterns,
URLs, host-name and personal-machine terms, landing/delivery terms, and
contribution refusal phrases. `git log --all --diff-filter=A --format='%H' -- PATH`
identified introductions; `git show` inspected context and removals. Package
lockfiles were excluded from the broad URL inventory as third-party dependency
metadata, not rewritten. Searches of the working README and tracked tree were
also checked after the edit. The local lineup override is the sole runner-path
exclusion; no other tracked path is excluded from that search.
