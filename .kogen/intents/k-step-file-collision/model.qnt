module k_step_file_collision {
  var legacy: bool
  var shared: bool
  var changed: bool
  var approved: bool
  var allowed: bool

  action init = all {
    legacy' = false, shared' = false, changed' = false,
    approved' = false, allowed' = false
  }
  action step = nondet l = Set(false, true).oneOf()
    nondet s = Set(false, true).oneOf()
    nondet c = Set(false, true).oneOf()
    nondet a = Set(false, true).oneOf()
    all {
      legacy' = l, shared' = s, changed' = c, approved' = a,
      allowed' = if (s) l and a and not(c) else not(c)
    }
  val NoNewSharing = shared and allowed implies legacy and approved
  val NoChangedDependencies = allowed implies not(changed)
  val PreserveLegacy = legacy and shared and approved and not(changed) implies allowed
}
