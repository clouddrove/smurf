---
name: smurf
description: Complete command reference and testing workflow for the smurf CLI (sdkr/Docker, selm/Helm, stf/Terraform). Use this whenever working in the smurf repo, running or testing any smurf command, adding a subcommand, verifying a release, or answering "what commands does smurf have" and "how do I test this". Also use before cutting a tag, since it lists exactly which checks must pass and which paths CI does not cover.
---

# smurf

A Go CLI wrapping Docker, Helm and Terraform behind one interface. It ships three
ways, and each has its own failure modes: a binary, a GitHub Action
(`action.yml`), and a Docker image bundling docker-cli, aws-cli, gcloud,
terraform and trivy.

**59 subcommands**: sdkr 15, selm 21, stf 17, plus 6 top-level.

## Build and run

```bash
make                       # build ./smurf with version ldflags
make install               # go install with the same ldflags
go run . <subcommand>      # run without building
make image                 # build the Docker image
make docs                  # regenerate docs/sm/docs/cli from the live commands
```

Regenerate docs after changing any command's `Use`, `Short`, `Long` or
`Example`. The CLI reference is generated, so hand-edits are lost; a stale
reference is easy to miss because nothing fails, it just goes out of date.

## Configuration: smurf.yaml

`smurf init` writes a starter file. The contract is that **when a flag or env
var is missing at runtime, the command falls back to the `smurf.yaml` value**.
That is a user-facing feature, not an internal convenience, so preserve it when
touching flag handling.

```yaml
sdkr:
  docker_username / docker_password
  github_username / github_token
  provisionAcrRegistryName / provisionAcrResourceGroup / provisionAcrSubscriptionID
  provisionGcrProjectID / google_application_credentials
  imageName / targetImageTag / dockerfile
  awsAccessKey / awsSecretKey / awsRegion
  awsECR / dockerHub / ghcrRepo / gcpRepo   # booleans selecting the registry
selm:
  deployHelm / releaseName / namespace / chartName / fileName / revision
```

## Command reference

### Top level

| Command | Usage |
|---|---|
| `init` | `smurf init` — write a starter smurf.yaml |
| `version` | `smurf version` — version, commit, build date, Go version, OS/arch |
| `deploy` | `smurf deploy` — build, push, then optional Helm deploy, driven entirely by smurf.yaml |
| `sdkr` / `selm` / `stf` | group commands, below |

### sdkr — Docker (15)

Uses the Docker Go SDK, not the `docker` binary. Every client is built with
`client.WithAPIVersionNegotiation()`; omit it and the client fails against any
daemon older than the SDK's default API version.

| Command | Usage |
|---|---|
| `sdkr init` | `smurf sdkr init` |
| `sdkr build` | `smurf sdkr build [IMAGE[:TAG]]` — **one** positional arg, not name and tag separately |
| `sdkr scan` | `smurf sdkr scan [IMAGE[:TAG]]` — trivy |
| `sdkr tag` | `smurf sdkr tag [SOURCE[:TAG]] [TARGET[:TAG]]` |
| `sdkr remove` | `smurf sdkr remove [IMAGE[:TAG]]` |
| `sdkr push hub\|aws\|az\|gcp` | `smurf sdkr push hub [IMAGE[:TAG]]` |
| `sdkr provision-hub\|-acr\|-ecr\|-gcp\|-ghcr` | build + scan + push for one registry |

Useful build flags: `--file`, `--context`, `--no-cache`, `--build-arg`,
`--target`, `--platform`, `--buildkit`, `--timeout`.

### selm — Helm (21)

Uses the Helm Go SDK. `internal/helm/release.go` centralises action config,
`error.go` normalises Helm errors.

| Command | Usage |
|---|---|
| `selm create` | `smurf selm create [NAME]` |
| `selm install` | `smurf selm install [RELEASE] [CHART]` |
| `selm upgrade` | `smurf selm upgrade [NAME] [CHART]` |
| `selm rollback` | `smurf selm rollback [RELEASE] [REVISION]` |
| `selm uninstall` | `smurf selm uninstall [NAME]` |
| `selm lint` | `smurf selm lint [CHART]` |
| `selm template` | `smurf selm template [RELEASE] [CHART]` |
| `selm list` / `status` / `history` | `smurf selm status [NAME]` |
| `selm pull` | `smurf selm pull [CHART]` |
| `selm repo add\|update\|debug` | `smurf selm repo add [NAME] [URL]` |
| `selm plugin install\|list\|uninstall` | `smurf selm plugin install [PLUGINS]` |
| `selm provision` | install + upgrade + lint + template |
| `selm init` | `smurf selm init` |

`--timeout` defaults differ per command: install 600s, rollback 300s, upgrade
120s. Each is bound to its **own** variable. They were once bound to a shared
package global, which meant pflag wrote each default into the same variable at
registration time and the last `init()` won, so all three silently used 120s.
Keep them separate.

### stf — Terraform (17)

Calls Terraform through `tfexec` / `hashicorp/terraform-exec`.

| Command | Usage |
|---|---|
| `stf init` / `validate` / `fmt` | `smurf stf fmt` — the command is **`fmt`**, not `format` |
| `stf plan` | `smurf stf plan` — `--detailed-exitcode` gives 0 none / 1 error / 2 pending |
| `stf apply` | `smurf stf apply [plan-file]` |
| `stf destroy` / `refresh` / `drift` / `graph` | `smurf stf drift` |
| `stf output` / `show` | `smurf stf show [plan-file]` |
| `stf import` | `smurf stf import ADDRESS ID` |
| `stf state-list\|-pull\|-push\|-rm` | `smurf stf state-rm [address...]` |
| `stf provision` | init + plan + apply + output |

## AI assistance

Every command takes `--ai`. On failure the error is redacted, truncated and
sent for a short explanation, which is then rendered above the raw error.

Any OpenAI-compatible endpoint works, which covers running at no cost:

```bash
# free and local, no key, offline
export OPENAI_BASE_URL=http://localhost:11434/v1
export OPENAI_MODEL=llama3.2

# free hosted tier
export OPENAI_BASE_URL=https://openrouter.ai/api/v1
export OPENAI_API_KEY=<key>
export OPENAI_MODEL=cohere/north-mini-code:free

# default: OpenAI
export OPENAI_API_KEY=sk-...
```

| Variable | Effect |
|---|---|
| `OPENAI_API_KEY` | credential; not required when the endpoint is local |
| `OPENAI_BASE_URL` | any OpenAI-compatible endpoint; unset means OpenAI |
| `OPENAI_MODEL` | model id; defaults to gpt-4o-mini |
| `SMURF_AI_NO_CACHE` | set to disable the response cache |
| `SMURF_AI_CACHE_TTL` | seconds or a duration; defaults to 24h |

Responses are cached on disk under the user cache dir, keyed by model,
endpoint and prompt, mode 0600 because an explanation embeds the error it came
from. A repeated failure is served locally: measured at 10.4s against
OpenRouter versus 237ms from cache.

Requests are capped at 700 tokens with temperature 0.2, and error text is cut
to 6000 characters keeping the tail, since the cause of a Terraform or Helm
failure sits at the end of its output.

**Known gap.** `--ai` is accepted by 42 internal functions and 19 of them have
error paths that never reach it, three that never fire at all. `smurf stf
validate --ai` is the clearest case: it prints nothing on a validation
failure. Check the path before assuming the flag works for a given command.

## GitHub Actions output

`internal/ci` writes the two things Actions understands, and both are no-ops
elsewhere so a terminal sees no change:

- workflow commands (`::error::`), which attach a message to the run and the
  pull request rather than only the log
- the job summary, markdown rendered at the top of the run

A failed `selm upgrade` publishes a table of unhealthy pods with their status
and reason, with the Helm error collapsed underneath, because Helm's own
message is usually "timed out waiting for the condition" and the pod reason is
what the reader wants. `internal/helm/cireport.go` holds this;
`ReportFailureToCI` publishes only the first failure in a process, so the
command layer and `HelmUpgrade` can both call it without reporting twice.

Escaping is not cosmetic: a raw newline ends a workflow command and would drop
everything after the first line, and an unescaped pipe shifts every later
column of a table.

## Architecture

Three layers, kept separate: **`cmd/` holds no business logic, `internal/`
imports no Cobra.**

1. `main.go` blank-imports the three subcommand packages so their `init()`
   registers with `cmd.RootCmd`.
2. `cmd/<group>/` — one file per subcommand, each registering itself in `init()`
   via `<group>Cmd.AddCommand(...)`.
3. `internal/<group>/` — SDK-backed implementations.

Shared flag variables live in `configs/types.go`. Prefer a command-local
variable when a flag's default differs per command; see the selm `--timeout`
note above for why.

### Adding a subcommand

1. Create the Cobra command in `cmd/<group>/`, register it in `init()`.
2. Put the real logic in `internal/<group>/`.
3. Bind genuinely shared flags to `configs/types.go`; keep per-command defaults local.
4. If it changes a primitive that a `provision*` meta-command chains, update that too.
5. Run `make docs`.
6. Output goes through `pterm` (`pterm.Success` / `Error` / `Info`), not `fmt.Println`.

Group commands (`sdkr`, `selm`, `stf`) declare `Args: cobra.NoArgs`. Without it
cobra treats an unknown subcommand as a positional argument, runs the group's
`Run` and exits **0** — so `smurf stf aply` reports success having done nothing.
Keep that guard.

## Testing

```bash
make test                        # unit tests
make test-integration            # integration suites (need real services)
go test -v ./test/docker -run TestBuild
make vet
./test/e2e/all-commands.sh ./smurf   # every subcommand
```

### Layers

| Layer | Location | Needs |
|---|---|---|
| Unit | `configs/`, `internal/*/` | nothing |
| CI output | `internal/ci` | nothing; set `GITHUB_STEP_SUMMARY` to a temp file to inspect |
| Command wiring | `test/helm/*_test.go` (untagged) | nothing |
| Integration | `test/{docker,helm,terraform}` — `//go:build integration` | Docker daemon, kind cluster, terraform binary |
| All subcommands | `test/e2e/all-commands.sh` | terraform + Docker for full coverage |

Integration suites carry a build tag and will **not** run under a plain
`go test ./...`. That is why they sat unexecuted for a long time; pass
`-tags integration`.

`all-commands.sh` discovers the command tree from the binary rather than a
hardcoded list, so a new subcommand is covered automatically. Tier 1 asserts
every command answers `--help`; tier 2 really runs what needs no cloud account.
It skips the docker and terraform tiers when those are unavailable, which keeps
it runnable anywhere but means **a local pass is weaker than a CI pass** — an
invalid docker invocation once passed locally and failed only in CI.

### What CI checks

| Workflow | Trigger | Covers |
|---|---|---|
| `build.yml` | push, PR | build and test |
| `lint.yml` | PR | golangci-lint |
| `go-security.yml` | push, PR | govulncheck gate, coverage |
| `integration.yml` | PR | the three integration suites + all subcommands |
| `action-check.yml` | PR | `action.yml` via `uses: ./` on ubuntu **and** macOS |
| `docker-check.yml` | PR | image build + every bundled tool runs |
| `docs-check.yml` | PR | `mkdocs build --strict` |
| `release-dry-run.yml` | PR | 6-way cross-compile, archives, changelog, checksums |
| `release.yml` | **tag push only** | the real release |
| `post-release-smoke.yml` | release published, or dispatch with a tag | the published assets, the ghcr image, and the action as published |

Keep `GO_VERSION` in the workflows, the `go` directive in `go.mod` and the
`golang:` tag in the Dockerfile aligned. When they drift, govulncheck evaluates
a different standard library than the one that ships.

The govulncheck gate (`.github/scripts/govulncheck_gate.py`) fails only on
symbol-reachable findings and needs `set -o pipefail` around it — govulncheck
exiting non-zero after emitting only its config header otherwise looks like a
clean scan.

### Before cutting a tag

`release.yml` runs only on tag push, so it is the one thing PRs cannot fully
rehearse. `release-dry-run.yml` covers the cross-compile matrix, archives,
changelog and checksums; the publish steps and the ghcr push are inherently
first exercised for real on the tag, since publishing is what they do.

`post-release-smoke.yml` catches the result within minutes. It runs when a
release publishes and checks what a user actually downloads: every published
archive against checksums.txt, all six platforms present, the linux binary
executed and reporting the tag, the ghcr image pulled anonymously with every
bundled tool run, and action.yml as published installing on ubuntu and macOS.

So the order is: PR CI green (which now includes the image build, all 59
subcommands and the release rehearsal), push the tag, watch the release run,
then confirm post-release smoke goes green rather than assuming it.

To check an existing release at any time:

```bash
gh workflow run "Post-release smoke" --ref master -f tag=v1.1.9
```

## Gotchas worth knowing

- `sdkr build` takes **one** argument in `IMAGE:TAG` form.
- The stf format command is `fmt`.
- `stf` sets `SilenceErrors: true`, so it exits 1 on an unknown subcommand but
  prints only its usage block, while sdkr and selm print `Error: unknown command`.
- macOS has no `sha256sum`; use `shasum -a 256`. `action.yml` runs on consumer
  runners including macOS, so anything added there needs both.
- `action.yml` resolves `version: latest` through the releases redirect, not the
  REST API — the API allows 60 unauthenticated requests per hour per IP and
  hosted runners share addresses.
- `--ai` being accepted does not mean it fires; see the known gap above.
- `internal/ci` is silent unless `GITHUB_ACTIONS=true` or `GITHUB_STEP_SUMMARY`
  is set, so to see its output locally, set them.
- Every artifact the repo downloads and then executes is pinned and
  checksum-verified. Bump the version and its hash together; they sit adjacent
  deliberately.
