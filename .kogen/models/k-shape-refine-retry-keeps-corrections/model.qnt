module retry {
  var phase: int
  var proposed: Set[str]
  var published: Set[str]
  action init = all { phase' = 0, proposed' = Set(), published' = Set() }
  action first = all { phase == 0, phase' = 1, proposed' = Set("A", "B", "C"), published' = published }
  action retry = all { phase == 1, phase' = 2, proposed' = proposed.union(Set("A")), published' = published }
  action finish = all { phase == 2, phase' = 3, proposed' = proposed, published' = proposed }
  action reject = all { phase > 0, phase < 3, phase' = 4, proposed' = proposed, published' = published }
  action stay = all { phase >= 3, phase' = phase, proposed' = proposed, published' = published }
  action step = any { first, retry, finish, reject, stay }
  val Retained = phase >= 1 implies proposed == Set("A", "B", "C")
  val Atomic = phase != 3 implies published == Set()
}
