# Contributing to Smurf

Thanks for your interest in contributing. This guide covers the development setup, project layout, and conventions for pull requests.

## Development setup

Requirements: Go 1.26 or newer, make, and git. Docker, Helm (a reachable cluster), and Terraform are only needed for integration tests.

```bash
git clone https://github.com/clouddrove/smurf.git
cd smurf
make            # builds ./smurf with version ldflags
make test       # unit tests, no daemons required
make vet        # go vet
```

Run the CLI during development with `go run . <subcommand>`, for example `go run . sdkr build --help`.

Integration tests need a running Docker daemon, a reachable Kubernetes cluster, and the terraform binary:

```bash
make test-integration
```

## Project layout

Three layers, kept strictly separate:

1. `main.go` wires the root command and blank-imports the subcommand packages.
2. `cmd/` holds Cobra command definitions only, one file per subcommand, grouped as `cmd/sdkr` (Docker), `cmd/selm` (Helm), `cmd/stf` (Terraform). No business logic here.
3. `internal/` holds the SDK-backed implementations (`internal/docker`, `internal/helm`, `internal/terraform`, plus `internal/ai` and `internal/utils`). Never import Cobra from `internal/`.

Shared configuration lives in `configs/`: the `smurf.yaml` schema (`configs/types.go`) and the loader with `${ENV_VAR}` interpolation (`configs/configs.go`). The contract: when a flag or environment variable is missing at runtime, commands fall back to the `smurf.yaml` value.

## Adding a subcommand

1. Create the Cobra command in `cmd/<group>/`, registered via `init()` with the group root.
2. Put the real logic in `internal/<group>/`.
3. Bind shared flags to package-level vars in `configs/types.go` only if they are read across packages.
4. Update the group's `provision` command if your change affects a step it chains.
5. Document the command in `docs/<group>/README.md` and, if user-facing behavior changed, the MkDocs pages under `docs/sm/docs/`.

## Conventions

- User-facing output goes through `pterm` (`pterm.Success`, `pterm.Error`, `pterm.Info`), not bare `fmt.Println`.
- Return errors from `RunE`; do not call `os.Exit` inside command bodies.
- Wrap errors with `%w` so callers can use `errors.Is` and `errors.As`.
- Destructive operations must not run without confirmation or an explicit flag (`--auto-approve`, `--yes`).
- New code that can be unit tested without a live daemon should come with tests. Integration tests go under `test/` behind the `integration` build tag.
- Format with `gofmt`; CI fails on unformatted code.

## Commits and pull requests

- Use conventional commit subjects: `fix(selm): ...`, `feat(stf): ...`, `docs: ...`, `ci: ...`.
- Reference related issues in the commit body or PR description.
- Keep PRs focused; unrelated changes belong in separate PRs.
- Make sure `make test`, `make vet`, and `gofmt -l .` are clean before opening the PR.

## Reporting bugs and requesting features

Open an issue at https://github.com/clouddrove/smurf/issues with reproduction steps (for bugs) or the use case (for features). For security vulnerabilities, do not open a public issue; see [SECURITY.md](SECURITY.md).
