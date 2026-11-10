import re,json,subprocess,pathlib
s=pathlib.Path('shape/shape.go').read_text()
for name in ['converseExternal','answer','parseAnswers','recommendedAnswers','writeQuestions','printQuestions','probe','validateBlocked','validateQuestions','optionProblemsOf']:
 m=re.search(r'(?m)^func (?:\([^\n]*?\) )?'+name+r'\([^\n]*\{',s)
 e=m.end()+re.search(r'(?m)^(?:func |type |const |var )',s[m.end():]).start()
 s=s[:m.end()]+' panic("outside bounded native controls") }\n\n'+s[e:]
s=re.sub(r'(?m)^\s*//.*\n','',s);s=re.sub(r'(?m)^\s+','',s)
m=re.search(r'func \(sh \*shaping\) fixAttempt[^\n]*\{',s);e=m.end()+re.search(r'(?m)^type submission',s[m.end():]).start();s=s[:m.end()]+' return nil }\n'+s[e:]
s=s.replace('defer turns.counts(sh.ev)','defer func(){ turns.countsBound(sh.ev, sh.requests) }()')
s=s.replace('sh.requests++','if sh.requests >= shapeRequestCap { return requestCapStop(last, lastAnswer), usage, nil }; sh.requests++',1)
s=s.replace('if noToolStreak >= noToolLimit {','if toolFreeExhausted(sinceSubmit) {')
s=s.replace('"github.com/KogenAI/kogen-coordinator/durable"','')
helper='''package shape
import "fmt"
const shapeRequestCap = 300
func toolFreeExhausted(count int) bool { return count >= noToolLimit }
func requestCapStop(last lastSubmission, answer string) outcome {
 o:=noToolStop(last,answer)
 o.detail=fmt.Sprintf("the shaper reached the provider request cap of %d",shapeRequestCap)
 if answer!="" { o.detail += "; last tool-free answer: " + answer }
 o.problems=append([]string{o.detail},o.problems...)
 return o
}
func(t *turnCounter) countsBound(ev *Evidence, count int) {
 ev.Record(map[string]any{"event":"shaping_turn_counts","owner":t.owner,"no_tool_turns":t.noTool,"invalid_submissions":t.invalid,"request_cap":shapeRequestCap,"requests":count})
}
'''
pathlib.Path('shape/shape.go').write_text(s);pathlib.Path('shape/toolfree.go').write_text(helper)
pathlib.Path('shape/bounds_acceptance_test.go').write_text(pathlib.Path('features/steps/shape-bounds-tool-free-answers__driver/cases.go').read_text())
r=subprocess.run(['go','test','./shape','-run','^TestBounds','-count=1'],capture_output=True,text=True)
if r.returncode: print(r.stdout+r.stderr);raise SystemExit(r.returncode)
correct={'shape/shape.go':s,'shape/toolfree.go':helper}
wrong=[]
for name,case,old,new in [('read resets count','alternating','if toolFreeExhausted(sinceSubmit) {','if noToolStreak >= noToolLimit {'),('submissions do not reset','reset','sinceSubmit = 0','sinceSubmit += 0'),('cap off by one','cap','sh.requests >= shapeRequestCap','sh.requests > shapeRequestCap')]:
 wrong.append({'name':name,'scenario':case,'step':'Then shape-bounds-tool-free-answers: '+case+' obeys the bound','files':{'shape/shape.go':s.replace(old,new)}})
assert sum(len(x.encode()) for x in correct.values())<=65536,[(k,len(v)) for k,v in correct.items()]
print(json.dumps({'correct':{'files':correct},'wrong':wrong}))
