module model {
  var refused: bool
  var approved: bool
  var hashMatches: bool
  action init = all { refused' = false, approved' = false, hashMatches' = true }
  action notReady = all { refused' = true, approved' = false, hashMatches' = hashMatches }
  action edit = all { refused' = refused, approved' = approved, hashMatches' = false }
  action successfulShape = all { refused' = false, approved' = false, hashMatches' = true }
  action approve = all { refused' = refused, approved' = (not(refused) and hashMatches), hashMatches' = hashMatches }
  action step = any { notReady, edit, successfulShape, approve }
  val RefusedNeverApproved = refused implies not(approved)
}
