module repeatControl {
  var cached: bool
  var packageVersion: int
  var treeVersion: int
  var cachedPackage: int
  var cachedTree: int
  var checks: int
  var runs: int
  var events: int
  var lastWasHit: bool
  var lastRunDelta: int
  var lastEventDelta: int

  action init = all {
    cached' = false, packageVersion' = 0, treeVersion' = 0,
    cachedPackage' = -1, cachedTree' = -1,
    checks' = 0, runs' = 0, events' = 0,
    lastWasHit' = false, lastRunDelta' = 0, lastEventDelta' = 0,
  }
  action check(outcome: str): bool = {
    val hit = cached and cachedPackage == packageVersion and cachedTree == treeVersion
    val runDelta = if (hit) 0 else 1
    val eventDelta = if (hit or outcome == "error") 0 else 1
    all {
      checks' = checks + 1, runs' = runs + runDelta, events' = events + eventDelta,
      lastWasHit' = hit, lastRunDelta' = runDelta, lastEventDelta' = eventDelta,
      cached' = (hit or outcome == "completed"),
      cachedPackage' = if (hit) cachedPackage else if (outcome == "completed") packageVersion else -1,
      cachedTree' = if (hit) cachedTree else if (outcome == "completed") treeVersion else -1,
      packageVersion' = packageVersion, treeVersion' = treeVersion,
    }
  }
  action changePackage = all {
    packageVersion' = packageVersion + 1, treeVersion' = treeVersion + 1,
    cached' = false, cachedPackage' = cachedPackage, cachedTree' = cachedTree,
    checks' = checks, runs' = runs, events' = events,
    lastWasHit' = false, lastRunDelta' = 0, lastEventDelta' = 0,
  }
  action changeTree = all {
    packageVersion' = packageVersion, treeVersion' = treeVersion + 1,
    cached' = false, cachedPackage' = cachedPackage, cachedTree' = cachedTree,
    checks' = checks, runs' = runs, events' = events,
    lastWasHit' = false, lastRunDelta' = 0, lastEventDelta' = 0,
  }
  action step = any { check("completed"), check("skipped"), check("error"), changePackage, changeTree }
  val ReuseWithoutDuplicate = lastWasHit implies (lastRunDelta == 0 and lastEventDelta == 0)
  val FreshKeyOnly = cached implies (cachedPackage == packageVersion and cachedTree == treeVersion)
  val ObservationStillRuns = runs <= checks and events <= runs
}
