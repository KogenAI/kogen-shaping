module model {
  var firstA: int
  var firstB: int
  var submission: int
  var note: int
  var expected: int
  var refused: bool
  action init = all { firstA' = 0, firstB' = 0, submission' = 0, note' = 0, expected' = 0, refused' = false }
  action checkA = all {
    firstA' = if (firstA == 0) submission + 1 else firstA,
    firstB' = firstB, submission' = submission + 1,
    note' = firstA, expected' = firstA, refused' = true
  }
  action checkB = all {
    firstB' = if (firstB == 0) submission + 1 else firstB,
    firstA' = firstA, submission' = submission + 1,
    note' = firstB, expected' = firstB, refused' = true
  }
  action pass = all { firstA' = firstA, firstB' = firstB, submission' = submission + 1, note' = 0, expected' = 0, refused' = false }
  action step = any { checkA, checkB, pass }
  val NotesOnlyForEarlierRefusals = (note == expected) and (note == 0 or (refused and note < submission))
}
