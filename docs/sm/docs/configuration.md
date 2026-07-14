# smurf.yaml Configuration Reference

`smurf.yaml` is the shared configuration file read by `sdkr`, `selm`, and `smurf deploy`. Its structure is defined by the `Config` struct in `configs/types.go`. If a required flag or environment variable is missing at runtime, the command falls back to the matching value in this file.

> **Security warning**
> Never commit `smurf.yaml` to version control once it holds real credentials (Docker Hub tokens, GitHub tokens, AWS keys, Azure subscription/resource-group IDs, GCP service-account paths). Prefer environment variables, or the `${ENV_VAR}` interpolation described below, over plaintext secrets. `smurf init`, `smurf sdkr init`, and `smurf selm init` all create the file with permissions `0600` (owner read/write only) precisely because it can hold secrets, and all three refuse to run if `smurf.yaml` already exists, so they never silently overwrite your configuration.

## `${ENV_VAR}` interpolation

Every string field below is expanded against the process environment before use:

- Only the braced form `${VAR_NAME}` is recognized. Bare `$VAR_NAME` is left as a literal string.
- A referenced variable that is not set expands to an empty string.
- Any other `$` in the value (for example inside a password like `P@ss$word123`) is left untouched.

```yaml
sdkr:
  docker_password: "${DOCKER_PASSWORD}"
  awsAccessKey: "${AWS_ACCESS_KEY_ID}"
```

## `sdkr` section (`SdkrConfig`)

| Field (YAML key) | Type | Purpose |
|---|---|---|
| `docker_password` | string | Docker Hub password used by `sdkr push hub` / `provision-hub` when `DOCKER_PASSWORD` is not already set in the environment. |
| `docker_username` | string | Docker Hub username, same fallback behavior as `docker_password`. |
| `github_username` | string | GitHub username for GHCR auth (`GITHUB_USERNAME` fallback), used by `provision-ghcr` and `smurf deploy`. |
| `github_token` | string | GitHub personal access token with `write:packages` scope, used for GHCR auth (`GITHUB_TOKEN` fallback). |
| `provisionAcrRegistryName` | string | Azure Container Registry name, used by `provision-acr` when `--registry-name` is not passed. |
| `provisionAcrResourceGroup` | string | Azure resource group containing the registry, used by `provision-acr` when `--resource-group` is not passed. |
| `provisionAcrSubscriptionID` | string | Azure subscription ID, used by `provision-acr` when `--subscription-id` is not passed. |
| `provisionGcrProjectID` | string | GCP project ID, used as a fallback by `push gcp` when `--project-id` is not passed and no image argument is given (`provision-gcp` requires `--project-id` explicitly for short image names). |
| `google_application_credentials` | string | Path to a GCP service-account JSON key file; exported as `GOOGLE_APPLICATION_CREDENTIALS` if that variable is not already set. |
| `imageName` | string | Image name (optionally `name:tag`) used by `build`, `push`, and all `provision-*` commands when no image argument is given. |
| `targetImageTag` | string | Default target tag used as a fallback by `sdkr tag` when no target argument is given. |
| `awsAccessKey` | string | Reserved for AWS access key ID. Currently only interpolated by `smurf init`/`sdkr init`; no command reads it back. AWS auth for ECR uses the standard AWS SDK credential chain (`AWS_ACCESS_KEY_ID`/`AWS_SECRET_ACCESS_KEY`/`AWS_REGION` env vars, shared config, or IAM role) instead. |
| `awsSecretKey` | string | Reserved for AWS secret access key; same caveat as `awsAccessKey` above. |
| `awsRegion` | string | Reserved for AWS region; same caveat as `awsAccessKey` above. |
| `dockerfile` | string | Reserved for a Dockerfile path. Currently only interpolated; no command reads it back. Use the `--file`/`-f` flag (or its default of `Dockerfile` in the build context) instead. |
| `awsECR` | bool | When `true`, `smurf deploy` pushes to AWS ECR. |
| `dockerHub` | bool | When `true`, `smurf deploy` pushes to Docker Hub. |
| `ghcrRepo` | bool | When `true`, `smurf deploy` pushes to GitHub Container Registry. |
| `gcpRepo` | bool | When `true`, `smurf deploy` pushes to GCP (GCR or Artifact Registry). |

Only one of `awsECR` / `dockerHub` / `ghcrRepo` / `gcpRepo` should be `true` at a time; `smurf deploy` picks the first matching registry in that order.

## `selm` section (`SelmConfig`)

| Field (YAML key) | Type | Purpose |
|---|---|---|
| `deployHelm` | bool | When `true`, `smurf deploy` installs or upgrades the Helm release after the image push completes. |
| `releaseName` | string | Helm release name used by `smurf deploy` (defaults to the chart's base name if empty). |
| `namespace` | string | Kubernetes namespace for the release (defaults to `default` if empty). |
| `chartName` | string | Path to the Helm chart to install/upgrade. |
| `fileName` | string | Path to a values file to apply; if empty, `smurf deploy` looks for `values.yaml` next to the chart. |
| `revision` | int | Revision number used as the fallback for `smurf selm rollback` when no `[REVISION]` argument is given. Not string-interpolated (it is an integer field). |

## Complete annotated example

```yaml
sdkr:
  docker_username: "my-docker-username"
  docker_password: "${DOCKER_PASSWORD}"          # prefer env interpolation over plaintext
  github_username: "my-github-username"
  github_token: "${GITHUB_TOKEN}"
  provisionAcrRegistryName: "myacrregistry"
  provisionAcrResourceGroup: "my-resource-group"
  provisionAcrSubscriptionID: "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  provisionGcrProjectID: "my-gcr-project-id"
  google_application_credentials: "/path/to/service-account-key.json"
  imageName: "my-application"
  targetImageTag: "v1.0.0"
  awsAccessKey: "${AWS_ACCESS_KEY_ID}"
  awsSecretKey: "${AWS_SECRET_ACCESS_KEY}"
  awsRegion: "us-east-1"
  dockerfile: "Dockerfile"
  awsECR: false
  dockerHub: false
  ghcrRepo: false
  gcpRepo: false
selm:
  deployHelm: false
  releaseName: "my-release"
  namespace: "my-namespace"
  chartName: "./charts/my-app"
  fileName: ""
  revision: 0
```

Run `smurf init` to scaffold this file (both sections at once, 0600, refuses to overwrite an existing `smurf.yaml`), or `smurf sdkr init` / `smurf selm init` to scaffold only one section.
