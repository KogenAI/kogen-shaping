# kogen-shaping

Kogen's human-assisted shaping workflow: `kogen intent shape`, `refine` and `approve`. Shaper is the human's role.

It is a component of [Kogen](https://github.com/KogenAI/kogen), which pins an exact version of it and assembles the `kogen` program.

## Getting started

Install Go 1.27.2 or later, Git, Node.js and npm. From a public clone:

```sh
git clone https://github.com/KogenAI/kogen-shaping.git
cd kogen-shaping
go build ./...
```

For `go test ./...`, first install the integration-test tools as described below.

Shaping helps a human turn a request into an Intent; refinement revises that Intent,
and approval is the human decision to authorize it. Kogen uses this component for
that workflow before building the approved work.

## Packages

- `shape`: shaping, refinement, approval cards, context windows and Intent contracts, and the Shaper's and Refine's model-facing policy: their prompts (`shape/prompts`) and tool definitions.
- `consumes`: one test suite per Kogen dependency (`consumes/<producer>/`) stating exactly what this repository relies on from it, each with a README inventory.

## Standalone Shape (developer entry point)

Build this repository's module without assembling Kogen:

```sh
go build -o kogen-shape ./cmd/kogen-shape
./kogen-shape --project /example/checkout --slug my-intent --request /example/request.txt
# Or: printf 'Request text' | ./kogen-shape --project /example/checkout --slug my-intent --request -
```

The checkout needs Shape's existing project configuration, readiness tools and provider
credentials; this command does not adopt or configure it, download tools, authenticate, approve,
or bypass readiness. Set `KOGEN_NODE`, `KOGEN_QUINT`, and `KOGEN_CUCUMBER` to the existing
Node executable, Quint installation and Cucumber installation respectively, as required by the
checkout's configuration. The caller's environment and working directory are used. The slug
must contain 3–48 lowercase letters, digits or dashes, and the UTF-8 request must not be empty.

Stdout contains exactly one JSON object and a newline; Shape's progress and questions go to
stderr. The process status matches `exit_code`. On publication, `intent` contains the absolute
`path` and exact `content` of the written `intent.md`, and `error` is null:

```json
{"exit_code":0,"intent":{"path":"/example/checkout/.kogen/intents/my-intent/intent.md","content":"..."},"error":null}
```

For a nonzero exit, `intent` is null and `error` is a diagnostic string. Usage or input errors
exit 2; Shape's own exit codes are preserved. In particular, exit 5 can mean pending questions
or an existing Intent. For pending questions, rerun the same command with the answers in the
request file or on stdin (for example `printf '1: a\n' | ./kogen-shape --project /example/checkout --slug my-intent --request -`).
The existing Shape question and answer semantics apply. Standalone Refine is not provided.

## Testing

The Go commands build and test the packages. Some integration tests run real
Cucumber and Quint and require Node.js and npm. Install their dependencies first:

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

For inquiries, contact contact@kogen.dev.

## License

Apache License 2.0. See [LICENSE](LICENSE) and [NOTICE](NOTICE).
