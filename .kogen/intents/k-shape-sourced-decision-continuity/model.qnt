// Sourced decisions across the runs of one shaping. Each run is a round. Within a run the shaper
// records a decision on evidence this round read or ran, or on a receipt an earlier round
// holds; cites unread or failed evidence (refused); asks a question a standing decision settles
// (refused) or an open one (the run ends with it and the next run continues); revises a standing
// decision with or without evidence this round read or ran; or submits an Intent that carries
// every standing decision, or one that omits one (refused). Refusals count against the existing
// submission limit of a run.
module ledger {
  pure val RetryLimit = 3
  pure val MaxRound = 3
  pure val MaxDecisions = 4

  type Decision = { round: int, receipts: Set[int], revises: int, standing: bool }

  var round: int
  var decisions: int -> Decision
  var invalid: int
  // "running", "asked" (an open question ended the run), "published" or "stopped"
  var phase: str
  var publishedWith: Set[int]

  def ids = decisions.keys()
  def standingIds = ids.filter(id => decisions.get(id).standing)
  def nextId = ids.size() + 1
  def receiptRounds = ids.fold(Set(), (acc, id) => acc.union(decisions.get(id).receipts))

  action init = all {
    round' = 1,
    decisions' = Map(),
    invalid' = 0,
    phase' = "running",
    publishedWith' = Set(),
  }

  action refuse = all {
    invalid' = invalid + 1,
    phase' = if (invalid + 1 >= RetryLimit) "stopped" else "running",
    round' = round,
    decisions' = decisions,
    publishedWith' = publishedWith,
  }

  action recordFresh = all {
    phase == "running",
    nextId <= MaxDecisions,
    decisions' = decisions.put(nextId, { round: round, receipts: Set(round), revises: 0, standing: true }),
    round' = round,
    invalid' = invalid,
    phase' = phase,
    publishedWith' = publishedWith,
  }

  // A receipt an earlier round holds keeps that round.
  action recordCarried(r: int): bool = all {
    phase == "running",
    nextId <= MaxDecisions,
    receiptRounds.contains(r),
    decisions' = decisions.put(nextId, { round: round, receipts: Set(r), revises: 0, standing: true }),
    round' = round,
    invalid' = invalid,
    phase' = phase,
    publishedWith' = publishedWith,
  }

  action citeUnreadOrFailed = all { phase == "running", refuse }

  action askSettled = all { phase == "running", standingIds.size() > 0, refuse }

  action askOpen = all {
    phase == "running",
    phase' = "asked",
    round' = round,
    decisions' = decisions,
    invalid' = invalid,
    publishedWith' = publishedWith,
  }

  action reviseFresh(id: int): bool = all {
    phase == "running",
    standingIds.contains(id),
    nextId <= MaxDecisions,
    decisions' = decisions
      .set(id, { ...decisions.get(id), standing: false })
      .put(nextId, { round: round, receipts: decisions.get(id).receipts.union(Set(round)), revises: id, standing: true }),
    round' = round,
    invalid' = invalid,
    phase' = phase,
    publishedWith' = publishedWith,
  }

  action reviseWithoutEvidence(id: int): bool = all { phase == "running", standingIds.contains(id), refuse }

  action submitCarryingAll = all {
    phase == "running",
    phase' = "published",
    publishedWith' = standingIds,
    round' = round,
    decisions' = decisions,
    invalid' = invalid,
  }

  action submitOmitting = all { phase == "running", standingIds.size() > 0, refuse }

  // The caller answers; the continuation is a new round with the same ledger.
  action continue = all {
    phase == "asked",
    round < MaxRound,
    round' = round + 1,
    phase' = "running",
    invalid' = 0,
    decisions' = decisions,
    publishedWith' = publishedWith,
  }

  action ended = all {
    phase != "running",
    not(phase == "asked" and round < MaxRound),
    round' = round,
    decisions' = decisions,
    invalid' = invalid,
    phase' = phase,
    publishedWith' = publishedWith,
  }

  action step = any {
    recordFresh,
    nondet r = 1.to(MaxRound).oneOf()
    recordCarried(r),
    citeUnreadOrFailed,
    askSettled,
    askOpen,
    nondet id = 1.to(MaxDecisions).oneOf()
    reviseFresh(id),
    nondet id2 = 1.to(MaxDecisions).oneOf()
    reviseWithoutEvidence(id2),
    submitCarryingAll,
    submitOmitting,
    continue,
    ended,
  }

  // A run stops only at the submission limit, and never beyond it.
  val InvalidBounded = invalid <= RetryLimit
  val StoppedOnlyAtLimit = phase == "stopped" implies invalid == RetryLimit
  // No receipt is attributed to a round after the one that recorded its decision.
  val ReceiptsNotLater = ids.forall(id => decisions.get(id).receipts.forall(r => r <= decisions.get(id).round))
  // A revision cites evidence its own round read or ran.
  val RevisionCitesItsRound = ids.forall(id => decisions.get(id).revises > 0 implies decisions.get(id).receipts.contains(decisions.get(id).round))
  // A revised decision stays on record and no longer stands.
  val RevisedKept = ids.forall(id => decisions.get(id).revises > 0 implies (ids.contains(decisions.get(id).revises) and not(decisions.get(decisions.get(id).revises).standing)))
  // A published Intent carries exactly the decisions that stand.
  val PublishedCarriesStanding = phase == "published" implies publishedWith == standingIds
}
