import json, pathlib, subprocess, sys
root = pathlib.Path.cwd()
data = json.loads((root / '.kogen/intents/shaping-cucumber-special-char-hint/controls.json').read_text())
files = data['correct']['files'] if sys.argv[1] == 'good' else data['wrong'][0]['files']
originals = {p: (root / p).read_bytes() for p in files}
target = root / 'shape/zz_special_char_hint_acceptance_test.go'
try:
    for p, text in files.items():
        (root / p).write_text(text)
    target.write_bytes((root / 'features/steps/shaping-cucumber-special-char-hint__driver/probe_test.go.txt').read_bytes())
    result = subprocess.run(['go', 'test', './shape', '-run', '^TestSpecialCharHintAcceptance$', '-count=1'], stdout=subprocess.PIPE, stderr=subprocess.STDOUT)
    print('accepted' if result.returncode == 0 else 'rejected')
    sys.exit(0 if result.returncode == 0 else 1)
finally:
    target.unlink(missing_ok=True)
    for p, content in originals.items():
        (root / p).write_bytes(content)
