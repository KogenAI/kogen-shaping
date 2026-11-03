// Native Shape's answer loop with the recovery inference. Each step is the shaper's answer to
// one request: tool calls, a tool-free answer, a refused submission of a package (0 to 2; a
// package refused before is unchanged) or a submission whose readiness failures come back for
// acknowledgment. The next request is automatic, a recovery inference or forced to submit.
module recovery {
  pure val NoToolLimit = 3
  pure val RetryLimit = 3
  pure val Packages = 0.to(2)

  var requests: int
  var toolCalls: int
  var noTool: int
  var sinceSubmit: int
  var invalid: int
  var readiness: int
  // "none", "refused", "refused_unchanged" or "readiness"
  var last: str
  // the enforcement of the next request and of the last answered one: "auto", "recovery" or "forced"
  var pending: str
  var sent: str
  // a request that required submit was sent since the last submission
  var forcedSinceSubmit: bool
  var refusedPackages: Set[int]
  // "" while running, "no_tool" or "invalid_budget" once stopped
  var ended: str

  action init = all {
    requests' = 0, toolCalls' = 0, noTool' = 0, sinceSubmit' = 0, invalid' = 0, readiness' = 0,
    last' = "none", pending' = "auto", sent' = "auto", refusedPackages' = Set(), ended' = "",
    forcedSinceSubmit' = false,
  }

  action answerTools = all {
    ended == "",
    requests' = requests + 1, toolCalls' = toolCalls + 1, sent' = pending, pending' = "auto",
    noTool' = noTool, sinceSubmit' = sinceSubmit, invalid' = invalid, readiness' = readiness,
    last' = last, refusedPackages' = refusedPackages, ended' = ended,
    forcedSinceSubmit' = (forcedSinceSubmit or pending == "forced"),
  }

  action answerProse = all {
    ended == "",
    requests' = requests + 1, toolCalls' = toolCalls, sent' = pending,
    noTool' = noTool + 1, sinceSubmit' = sinceSubmit + 1,
    ended' = if (sinceSubmit + 1 >= NoToolLimit) "no_tool" else "",
    pending' = if (sinceSubmit + 1 >= NoToolLimit) pending
      else if (sinceSubmit == 0 and (last == "refused" or last == "readiness")) "recovery"
      else "forced",
    invalid' = invalid, readiness' = readiness, last' = last, refusedPackages' = refusedPackages,
    forcedSinceSubmit' = (forcedSinceSubmit or pending == "forced"),
  }

  action submitRefused(pkg: int): bool = all {
    ended == "",
    requests' = requests + 1, toolCalls' = toolCalls, sent' = pending, pending' = "auto",
    noTool' = noTool, sinceSubmit' = 0, invalid' = invalid + 1, readiness' = readiness,
    last' = if (refusedPackages.contains(pkg)) "refused_unchanged" else "refused",
    refusedPackages' = refusedPackages.union(Set(pkg)),
    ended' = if (invalid + 1 >= RetryLimit) "invalid_budget" else "",
    forcedSinceSubmit' = false,
  }

  action submitReadinessFailures = all {
    ended == "", readiness < 2,
    requests' = requests + 1, toolCalls' = toolCalls, sent' = pending, pending' = "auto",
    noTool' = noTool, sinceSubmit' = 0, invalid' = invalid, readiness' = readiness + 1,
    last' = "readiness", refusedPackages' = refusedPackages, ended' = ended,
    forcedSinceSubmit' = false,
  }

  action stopped = all {
    ended != "",
    requests' = requests, toolCalls' = toolCalls, noTool' = noTool, sinceSubmit' = sinceSubmit,
    invalid' = invalid, readiness' = readiness, last' = last, pending' = pending, sent' = sent,
    refusedPackages' = refusedPackages, ended' = ended, forcedSinceSubmit' = forcedSinceSubmit,
  }

  action step = any {
    answerTools,
    answerProse,
    nondet pkg = Packages.oneOf()
    submitRefused(pkg),
    submitReadinessFailures,
    stopped,
  }

  // A recovery inference follows only the first tool-free answer after a refused submission of a
  // changed package or after readiness failures.
  val RecoveryOnlyAfterRepairableSubmission =
    pending == "recovery" implies (sinceSubmit == 1 and (last == "refused" or last == "readiness"))

  // An unchanged package's refusal never earns a recovery inference.
  val NoRecoveryAfterUnchanged = last == "refused_unchanged" implies pending != "recovery"

  // Before the no-tool stop, a request that required submit was sent since the last submission:
  // the recovery inference replaces only the first one.
  val NoToolStopFollowsForced = ended == "no_tool" implies forcedSinceSubmit

  // At most two tool-free answers between submissions, three before the no-tool stop.
  val ToolFreeAnswersBounded = {
    val segments = invalid + readiness + (if (ended == "invalid_budget") 0 else 1)
    noTool <= 2 * segments + (if (ended == "no_tool") 1 else 0)
  }

  // Prose and refused submissions alone end within nine requests at the default limits.
  val ProseAndRefusalsEndWithinNine =
    (toolCalls == 0 and readiness == 0) implies (requests <= 9 and (requests == 9 implies ended != ""))

  val RefusalsBounded = invalid <= RetryLimit
}
