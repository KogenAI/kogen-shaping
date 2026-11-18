# Revert fixer changes without forbidden Git

Slug: shape-fixer-revert-avoids-forbidden-git-guard
Model: not applicable — a file-revert step with no state machine; its rules are observed on real Git repositories
Acceptance:
- .kogen/intents/shape-fixer-revert-avoids-forbidden-git-guard/acceptance.feature
- features/steps/shape-fixer-revert-avoids-forbidden-git-guard__steps.js
- features/steps/shape-fixer-revert-avoids-forbidden-git-guard__driver/accept_fixer_revert_test.go.txt

The release acceptance suite fails the shaping scenario "Shaping formats the files it writes with the project's fixers and leaves other files alone" (spec/features/02-shaping.feature line 488) with `controller/internal_error: git command not allowed: git restore` (exit 70). The kogen-coordinator Git runner guard (kogen-coordinator main 11f62aa8, in the release tuple as e643661c) deliberately refuses `restore`, `clean`, `symbolic-ref`, `worktree` and `remote` through `Repo.Run`, `RunEnv`, `OK` and `OKEnv`, besides the commands it already refused. In `shape/shape.go` `fixAttempt`, after the fix commands run, a file that changed outside the shaped paths and exists at HEAD but was not changed before the fixers is put back with the call `sh.repo.OK("restore", "--source=HEAD", "--staged", "--worktree", "--", ":(literal)" + p)`; the guard refuses that call. kogen-shaping's own tests did not see it because go.mod pins the older coordinator d0dab3d6 (20261116), while the assembled kogen binary resolves the coordinator to e643661c. The guard stays as it is. This call is the one the change removes.

Expected behavior:

1. A tracked file that a fixer changed outside the shaped paths and that was clean before the fixers is returned to its HEAD content, HEAD mode and unstaged state, without sending `restore`, `reset`, `checkout`, `clean`, `symbolic-ref`, `worktree` or `remote` through the guarded runner. The Build writes the HEAD blob back with the file mode HEAD records (read the blob with `Repo.Show("HEAD", p)` and the mode and object id from the HEAD tree entry of that path, `git ls-tree -z HEAD -- p` through the runner or `Repo.TreeEntries("HEAD")`), and drops the staged change with `git update-index --add --cacheinfo <mode>,<object>,<path>` (or its three-argument form), which the guard allows.
2. A path that is a symbolic link (mode 120000) or an executable (100755) at HEAD keeps that type and mode after the revert: a link is re-created with `os.Symlink` to the HEAD target, an executable is written with mode 0755 (subject to the umask), after removing whatever the fixer left at the path.
3. The guard is not touched or weakened, and kogen-shaping calls no newly forbidden command through the guarded runner anywhere else. `cmd/shapebench` uses its own exec and is out of scope.
4. kogen-shaping's go.mod pins kogen-coordinator at e643661cb646 or newer (with go.sum), so its package tests run the guard that ships. If the module cannot be fetched in the Build, the Build states that, and builds the guard-bound test runner differently; it never leaves the tests on an older guard.
5. Everything else about the fixer pass stays: which files are reverted (a file already changed before the fixers is rewritten from its snapshot, a file that did not exist is removed), the progress lines and the shaping time line.

Tests the Build adds: model-free Go tests in the new file `shape/fixer_revert_test.go`, using the existing shape fixtures, run as `go test ./shape -run TestFixerRevert`: TestFixerRevertRestoresATrackedFile, TestFixerRevertKeepsModeAndLinks, TestFixerRevertSendsNoForbiddenGitCommand and TestFixerRevertKeepsPriorBehaviourForSnapshotted. Release evidence for the report: the Build cannot run the assembly suite; TestFixerRevertRestoresATrackedFile and TestFixerRevertSendsNoForbiddenGitCommand fail on main under the e643661c guard with `git command not allowed` and pass after the change.

No change to kogen-coordinator, kogen-setup, kogen-loop or the kogen CLI; no new flag, setting or environment variable.

## Decisions
- The revert writes the HEAD blob and mode to the worktree itself and re-indexes the same blob with `git update-index --add --cacheinfo`, because `update-index` is outside the guard's forbidden list while `restore`, `reset`, `checkout` and `clean` are refused; `read-tree` cannot take paths and would touch the whole index. (decision: D1, round 1) (evidence: shape/shape.go; go.mod) (assumption: the guard of e643661c still allows update-index and ls-tree; acceptance runs the real guard of e643661cb646 in a scratch copy, so a refusal shows there) (source: request)
- go.mod pins github.com/KogenAI/kogen-coordinator at v0.0.0-20261118001611-e643661cb646 and go.sum gets its hashes; the module resolved online when shaping inspected it (go list -m returned that version), so the Build updates it with go get; if the Build cannot reach the module, it says so and gives the tests a guard-bound runner another way. (decision: D2, round 1) (evidence: go.mod) (assumption: the Build environment can fetch the module; otherwise the fallback in the sentence applies) (source: request)
- Acceptance observes the behavior through probe tests of its own, run in a scratch copy of the checkout that is moved to the e643661cb646 guard, so it does not depend on test names the Build writes; the Build's own tests are required by name but acceptance only checks that the file declares them. (decision: D3, round 1) (evidence: shape/shape.go) (source: request)
- Files reverted, snapshot handling, progress lines and the time line of the fixer pass stay as they are; only the last branch's restore call changes. (decision: D4, round 1) (evidence: shape/shape.go) (source: request)

## Source comparison
- Addressed: expected 1 (tracked file back to HEAD content and unstaged), scenarios 1 and 3 of acceptance.feature.
- Addressed: expected 2 (executable and symbolic link keep mode and type), scenario 2.
- Addressed: expected 3 (no forbidden command through the runner, guard unchanged), scenario 3 records every git invocation of the pass through a PATH shim and also shows the guard refuses `restore`.
- Addressed: expected 4 (go.mod pin), scenario 5 reads go.mod; scenarios 1 to 4 run under the e643661cb646 guard.
- Addressed: expected 5 (prior behavior for snapshotted and new files), scenario 4.
- Addressed: tests in shape/fixer_revert_test.go with the four named functions, scenario 6.
- Superseded: the call `sh.repo.OK("restore", "--source=HEAD", "--staged", "--worktree", "--", ":(literal)" + p)` in shape/shape.go is removed.
- Omitted by request: cmd/shapebench, kogen-coordinator, kogen-setup, kogen-loop and the kogen CLI.

## Tests
- A tracked clean file a fixer changed outside the shaped paths returns to HEAD content and unstaged: TestFixerRevertRestoresATrackedFile in shape/fixer_revert_test.go (integration)
- An executable and a symbolic link keep HEAD mode and target after the revert: TestFixerRevertKeepsModeAndLinks in shape/fixer_revert_test.go (integration)
- The fixer pass sends no forbidden command through the guarded runner and the real pinned guard raises no error: TestFixerRevertSendsNoForbiddenGitCommand in shape/fixer_revert_test.go (integration)
- A file changed before the fixers gets its snapshot back and a new file is removed: TestFixerRevertKeepsPriorBehaviourForSnapshotted in shape/fixer_revert_test.go (integration)
- The shaping formats-with-fixers behavior is unchanged: TestShapeFormatsItsFilesWithTheFixersAndLeavesOthersAlone in shape/shape_test.go (existing)

## Feasibility
- none
