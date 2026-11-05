# Skip symlinks in external cited evidence

Slug: fix-k-shape-environment-stop-evidence
Model: not applicable — one added read-time check in `citedEvidence` with no state across calls
Acceptance:
- .kogen/intents/fix-k-shape-environment-stop-evidence/acceptance.feature
- features/steps/fix-k-shape-environment-stop-evidence__steps.js
- features/steps/fix-k-shape-environment-stop-evidence__harness.js

A review of the landed k-shape-environment-stop-evidence commit found this problem. For the external shaper, `citedEvidence` in `shape/question.go` shows a question's cited path when it is tracked at HEAD, does not end in `/`, and the session inspected something. To show it, `citedEvidence` reads `os.ReadFile(filepath.Join(sh.root, filepath.FromSlash(p)))`, which follows symlinks. As a result, a tracked symlink (git mode 120000) that points to a file outside the checkout has that outside file's contents sent to the question check model as `Question <n> cites <path> (read from the checkout):`. **This was reproduced on the unchanged tree (HEAD ffd3c15).** The repository was a copy of the shape tests' is-even project with `lib/weather.js` committed and `lib/secret.js` committed as an absolute symlink to a temp file containing `OUTSIDE-SECRET-7f3a: this file lies outside the checkout`. A question citing `(evidence: lib/weather.js; lib/secret.js)` sent this block to the question check: `Question 1 cites lib/secret.js (read from the checkout):\n1\tOUTSIDE-SECRET-7f3a: this file lies outside the checkout`.

**Fix (Lstat skip).** In the external branch of `citedEvidence` (`case sh.ext != nil:` after the "session read nothing" case), the code calls `os.Lstat` on the cited path under `sh.root` before reading it. If the path is a symbolic link (`fi.Mode()&fs.ModeSymlink != 0`), nothing is read, and the block is one line: `Question <n> cites <item as cited>: not shown (<path> is a symbolic link)`. Like the other "not shown" blocks, it uses `head+cited+": not shown ("+it.path+" is a symbolic link)"`. The item is not added to `shown`, so it never counts as a shown citation for the environment stop. All symlinks are skipped this way, whatever they point to. An `Lstat` error keeps today's behavior: the `os.ReadFile` failure falls back to `git show HEAD:<path>`, which for a symlink gives the link text, never the target's contents. Regular files are read and shown exactly as today.

Unchanged: the native shaper's path (it shows the text of this shaping's own `read_file` call, and that tool already refuses symlinks leaving the root); the command citations; the other "not shown" reasons and their order; the question check input when nothing is cited; the prompts and the policy goldens. The change makes no other edits.

## Decisions
- Reproduce first: the external shaper's question check received the contents of an outside file through a tracked symlink on the unchanged tree (evidence: ran `go test -c -vet=off -overlay <harness> ./shape` and the harness test, observed `Question 1 cites lib/secret.js (read from the checkout):\n1\tOUTSIDE-SECRET-7f3a: ...`) (source: request)
- The fix skips symlinks via `os.Lstat` instead of reading `git show HEAD:p`, so the external path still reads the checkout's text for regular files (source: request)
- A skipped symlink is reported with a new "not shown" reason, `<path> is a symbolic link`, in the existing one-line block form (source: request)
- Only the cited path itself is checked with Lstat. Git never tracks a file beneath a symlinked directory at HEAD, and the tracked-at-HEAD check stays in front (source: request)

## Source comparison
- addressed: "a tracked symlink to an outside file (e.g. /etc) has its contents sent to the check model" in `shape/question.go` `citedEvidence`. The bug was reproduced on main (see above). The acceptance scenario commits a symlink to an outside file and checks that its text reaches no session (`secret_sent: false`) and that the block is `not shown (lib/secret.js is a symbolic link)`.
- addressed: "First reproduce it: add a test that fails on current main for exactly this scenario" and "Add a symlink test". A symlink case is added to `TestExternalShapeQuestionCheckCitedEvidence` and must fail on the unfixed code (see Tests).
- addressed: "Skip symlinks via Lstat, or read `git show HEAD:p`". Lstat skip was chosen (Decisions).
- preserved: "Keep every existing test green; no unrelated changes". The existing cited-evidence cases keep their exact expected texts.

## Tests
- The external shaper does not read a cited tracked symlink and reports it as not shown because it is a symbolic link, while a regular tracked file cited beside it is still shown: add a case to `TestExternalShapeQuestionCheckCitedEvidence` in `shape/question_test.go` that commits `lib/secret.js` as a symlink to a marker file outside the checkout (under `t.TempDir()`), cites both files, asserts the exact cited-evidence text and the marker's absence, and fails on the unfixed code (regression)
- The external shaper does not send a tracked symlink's outside target to the question check, end to end: the acceptance scenario through the external Shape run (scenario)
- Regular tracked files, untracked paths and the session-read-nothing reason are unchanged: `TestExternalShapeQuestionCheckCitedEvidence` and `TestShapeQuestionCheckCitedEvidence` (existing)

## Feasibility
- These are scratch runs from this read-only external session, not Kogen probe records. The harness of `features/steps/fix-k-shape-environment-stop-evidence__harness.js` compiled into the shape tests through `go test -c -vet=off -overlay` and passed on the unchanged tree (about 1 s). It observed `tree_entry_mode` `120000`, exit 5, 2 sessions, `secret_sent: true`, and the cited evidence with the outside file's text. This is the reproduced fault, and the acceptance expects its fixed form.
- The observer controls run the step file's judgments. The good control uses the real `-test.v` output of that harness run plus the feature's expected observation. The bad control uses the unchanged tree's observed evidence, a failed, unfinished, failed-test, doubled, empty or observation-less run, and a wrong or missing value for every compared key.
- evidence: {"id":"symlink-evidence-observer","kind":"observer","inputs":["features/steps/fix-k-shape-environment-stop-evidence__steps.js","features/steps/fix-k-shape-environment-stop-evidence__harness.js",".kogen/intents/fix-k-shape-environment-stop-evidence/acceptance.feature"],"observers":["features/steps/fix-k-shape-environment-stop-evidence__steps.js"],"checks":[{"name":"good","command":"node features/steps/fix-k-shape-environment-stop-evidence__steps.js --control good","exit":0,"output":"accepted\n"},{"name":"bad","command":"node features/steps/fix-k-shape-environment-stop-evidence__steps.js --control bad","exit":1,"output":"rejected\n"}]}
