module toolfree {
 var requests: int
 var answers: int
 var stopped: bool
 action init = all { requests' = 0, answers' = 0, stopped' = false }
 action step = if (stopped) all { requests' = requests, answers' = answers, stopped' = stopped } else {
  nondet response = Set(0, 1, 2).oneOf()
  val nextAnswers = if (response == 2) 0 else if (response == 1) answers + 1 else answers
  all { requests' = requests + 1, answers' = nextAnswers, stopped' = (requests + 1 >= 300 or nextAnswers >= 3) }
 }
 val Bounded = requests <= 300 and answers <= 3
 val StopAtBoundary = (requests == 300 or answers == 3) implies stopped
}
