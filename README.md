# kogen-shaping

Kogen's human-assisted shaping workflow: `kogen intent shape`, `refine` and `approve`. Shaper is the human's role.

It is a component of [Kogen](https://github.com/KogenAI/kogen), which pins an exact version of it and assembles the `kogen` program.

## Packages

- `shape`: shaping, refinement, approval cards, audits, context windows and Intent contracts, and the Shaper's and Refine's model-facing policy: their prompts (`shape/prompts`) and tool definitions.

## Testing

Some tests run the real Cucumber and Quint. Install them once:

```sh
npm ci --prefix testdata/tools/cucumber
npm ci --prefix testdata/tools/quint
```

Then run:

```sh
go build ./... && go vet ./... && go test -race -count=1 ./...
```

After deliberate behavior changes or to resolve golden conflicts on the combined tree, regenerate
synthetic fixtures only (never live user data or providers):

```sh
go test ./... -update
```

Inspect the diff, then run `go test ./...` and the normal full checks above. Never hand-merge
hashes. The force golden is a deliberate compatibility baseline, not a routine snapshot. So are the
policy goldens (`shape/testdata/shaper_policy_golden.json`, `prompts_golden.json` and
`defs_golden.json`): they record the prompt and tool bytes the models have always received, and
change only with a deliberate, reviewed policy change.

The module depends on private KogenAI modules: set `GOPRIVATE=github.com/KogenAI/*`.

## License

Apache License 2.0. See [LICENSE](LICENSE) and [NOTICE](NOTICE).
