// The shaping conversation after a refused incomplete package: counted submissions, the free
// pre-check that follows a completeness refusal, tool-free answers and the run's end. Reads and
// proposals other than the one that completes the package change none of these counters.
module recovery {
  pure val LIMIT = 3
  pure val NO_TOOL = 3

  // The last counted refusal: "" none, "envelope" or "files" (a completeness refusal), "other".
  var lastGap: str
  // A free pre-check is pending: the next submit call is checked before it counts.
  var pending: bool
  // The accumulated package still lacks a declared file.
  var filesGap: bool
  var invalid: int
  var completeness: int
  var intercepts: int
  var sinceSubmit: int
  // Prose answers, submit calls and intercepted calls of the run.
  var turns: int
  // "" running, "ready", "blocked", "invalid_output", "no_tool".
  var ended: str

  def isCompleteness(g: str): bool = g == "envelope" or g == "files"

  action init = all {
    lastGap' = "",
    pending' = false,
    filesGap' = true,
    invalid' = 0,
    completeness' = 0,
    intercepts' = 0,
    sinceSubmit' = 0,
    turns' = 0,
    ended' = "",
  }

  action unchangedAll = all {
    lastGap' = lastGap, pending' = pending, filesGap' = filesGap, invalid' = invalid,
    completeness' = completeness, intercepts' = intercepts, sinceSubmit' = sinceSubmit,
    turns' = turns, ended' = ended,
  }

  action prose = all {
    lastGap' = lastGap, pending' = pending, filesGap' = filesGap, invalid' = invalid,
    completeness' = completeness, intercepts' = intercepts,
    sinceSubmit' = sinceSubmit + 1,
    turns' = turns + 1,
    ended' = if (sinceSubmit + 1 >= NO_TOOL) "no_tool" else "",
  }

  action proposeMissing = all {
    filesGap,
    lastGap' = lastGap, pending' = pending, filesGap' = false, invalid' = invalid,
    completeness' = completeness, intercepts' = intercepts, sinceSubmit' = sinceSubmit,
    turns' = turns, ended' = ended,
  }

  action intercept = all {
    lastGap' = lastGap, pending' = false, filesGap' = filesGap, invalid' = invalid,
    completeness' = completeness, intercepts' = intercepts + 1, sinceSubmit' = sinceSubmit,
    turns' = turns + 1, ended' = ended,
  }

  action refuse(gap: str): bool = all {
    lastGap' = gap,
    pending' = isCompleteness(gap),
    filesGap' = filesGap,
    invalid' = invalid + 1,
    completeness' = if (isCompleteness(gap)) completeness + 1 else completeness,
    intercepts' = intercepts,
    sinceSubmit' = 0,
    turns' = turns + 1,
    ended' = if (invalid + 1 < LIMIT) "" else if (isCompleteness(gap)) "blocked" else "invalid_output",
  }

  action accept = all {
    lastGap' = lastGap, pending' = false, filesGap' = filesGap, invalid' = invalid,
    completeness' = completeness, intercepts' = intercepts, sinceSubmit' = 0,
    turns' = turns + 1, ended' = "ready",
  }

  // A submit call whose envelope breaks the exactly-one rule.
  action submitEnvelope = if (pending) intercept else refuse("envelope")

  // A files submission: a declared file still missing is a completeness gap; a complete package
  // is accepted or refused at a later gate.
  action submitFiles = if (filesGap) { if (pending) intercept else refuse("files") }
    else any { accept, refuse("other") }

  action step = if (ended != "") unchangedAll else any {
    prose,
    proposeMissing,
    submitEnvelope,
    submitFiles,
  }

  // Every free pre-check follows its own counted completeness refusal.
  val InterceptsFollowRefusals = intercepts <= completeness
  // The attempt budget is not raised.
  val BudgetUnchanged = invalid <= LIMIT
  // A pre-check is pending only after a completeness refusal.
  val PendingOnlyAfterCompleteness = pending implies isCompleteness(lastGap)
  // At the budget, an incomplete package ends with the blocked report, never invalid output.
  val IncompleteBudgetEndsBlocked =
    (ended == "blocked" implies (invalid == LIMIT and isCompleteness(lastGap))) and
    ((invalid == LIMIT and isCompleteness(lastGap)) implies ended == "blocked")
  // Prose, submissions and pre-checks end within 11 turns at the default limits.
  val TurnsBounded = turns <= 11
}
