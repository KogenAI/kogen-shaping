Feature: Shaping reverts fixer changes without forbidden Git commands
  This feature runs probe tests of the fixer pass in a scratch copy of the project that uses the kogen-coordinator guard of e643661cb646 or newer.

  Scenario: A tracked file a fixer appended to is back at HEAD and unstaged
    Given shape-fixer-revert-avoids-forbidden-git-guard: a scratch copy of the project under the coordinator guard of e643661cb646
    When shape-fixer-revert-avoids-forbidden-git-guard: I run the probe test "TestAccFixerRevertTrackedFile"
    Then shape-fixer-revert-avoids-forbidden-git-guard: the probe test "TestAccFixerRevertTrackedFile" passed

  Scenario: A tracked executable and symbolic link keep their mode and target
    Given shape-fixer-revert-avoids-forbidden-git-guard: a scratch copy of the project under the coordinator guard of e643661cb646
    When shape-fixer-revert-avoids-forbidden-git-guard: I run the probe test "TestAccFixerRevertModeAndLink"
    Then shape-fixer-revert-avoids-forbidden-git-guard: the probe test "TestAccFixerRevertModeAndLink" passed

  Scenario: The fixer pass sends no forbidden Git command and the guard still refuses restore
    Given shape-fixer-revert-avoids-forbidden-git-guard: a scratch copy of the project under the coordinator guard of e643661cb646
    When shape-fixer-revert-avoids-forbidden-git-guard: I run the probe test "TestAccFixerRevertNoForbiddenCommand"
    Then shape-fixer-revert-avoids-forbidden-git-guard: the probe test "TestAccFixerRevertNoForbiddenCommand" passed

  Scenario: A file changed before the fixers gets its text back and a new file is removed
    Given shape-fixer-revert-avoids-forbidden-git-guard: a scratch copy of the project under the coordinator guard of e643661cb646
    When shape-fixer-revert-avoids-forbidden-git-guard: I run the probe test "TestAccFixerRevertSnapshotted"
    Then shape-fixer-revert-avoids-forbidden-git-guard: the probe test "TestAccFixerRevertSnapshotted" passed

  Scenario: The module pins the coordinator at e643661cb646 or newer
    When shape-fixer-revert-avoids-forbidden-git-guard: I read the coordinator pin of go.mod
    Then shape-fixer-revert-avoids-forbidden-git-guard: the pinned coordinator is at least 20261118001611-e643661cb646

  Scenario: The Build's tests live in shape/fixer_revert_test.go
    When shape-fixer-revert-avoids-forbidden-git-guard: I read the file "shape/fixer_revert_test.go"
    Then shape-fixer-revert-avoids-forbidden-git-guard: it declares the tests "TestFixerRevertRestoresATrackedFile, TestFixerRevertKeepsModeAndLinks, TestFixerRevertSendsNoForbiddenGitCommand, TestFixerRevertKeepsPriorBehaviourForSnapshotted"
