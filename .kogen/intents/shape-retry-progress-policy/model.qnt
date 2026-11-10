module retryProgress {
  var gate: int
  var consecutive: int
  var total: int
  var stopped: bool

  action init = all {
    gate' = 0,
    consecutive' = 0,
    total' = 0,
    stopped' = false,
  }

  action step = if (stopped) {
    all { gate' = gate, consecutive' = consecutive, total' = total, stopped' = stopped }
  } else {
    nondet nextGate = Set(1, 2, 3).oneOf()
    val nextCount = if (gate == nextGate) consecutive + 1 else 1
    all {
      gate' = nextGate,
      consecutive' = nextCount,
      total' = total + 1,
      stopped' = (nextCount >= 3 or total + 1 >= 6),
    }
  }

  val Bounds = consecutive >= 0 and consecutive <= 3 and total >= consecutive and total <= 6
  val StopPolicy = stopped == (consecutive >= 3 or total >= 6)
}
