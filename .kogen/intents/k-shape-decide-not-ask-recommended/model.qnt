module model {
  var phase: str
  var pending: bool
  var assumed: bool
  action init = all { phase' = "active", pending' = false, assumed' = false }
  action recommend = all { phase == "active", phase' = "ready", pending' = false, assumed' = true }
  action ask = all { phase == "active", phase' = "waiting", pending' = true, assumed' = assumed }
  action abortSession = all { phase != "ready", phase' = "failed", pending' = false, assumed' = assumed }
  action restart = all { phase == "failed", phase' = "active", pending' = false, assumed' = false }
  action stay = all { phase' = phase, pending' = pending, assumed' = assumed }
  action step = any { recommend, ask, abortSession, restart, stay }
  val NoStale = phase == "failed" implies not(pending)
  val RecommendedIsAssumed = phase == "ready" implies (assumed and not(pending))
}
