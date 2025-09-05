### Using Smurf SELM on local
Use `smurf selm <command>` to run Helm commands. Supported commands include:

- **Help:** `smurf selm --help`
- **Create a Helm Chart:** `smurf selm create`
- **Install a Chart:** `smurf selm install`
- **Upgrade a Release:** `smurf selm upgrade`
- **Provision Helm Environment:** `smurf selm provision --help`

The `provision` command for Helm combines `install`, `upgrade`, `lint`, and `template`.

### Using Smurf SELM in GitHub Action
### This GitHub Action creates helm chart and renders chart templates.

```yaml
name: Smurf SELM Workflow

on:
  push:
    branches:
      - main

jobs:
  terraform:
    runs-on: ubuntu-latest
    steps:
      - name: Checkout repository
        uses: actions/checkout@v3

      - name: Setup Smurf
        uses: clouddrove/smurf@v0.0.4

      - name: Smurf selm create
        run: |
          smurf selm create test-smurf

      - name: Smurf selm Template
        run: |
          smurf selm template my-release ./test-smurf
```

### All available commands in Smurf SELM

| Command   | Description                          |
|-----------|--------------------------------------|
| `create`    | Create a new Helm chart in the specified directory |
| `install` | Install a Helm chart into a Kubernetes cluster         |
| `lint`    | Lint a Helm chart |
| `list`   | List all Helm releases                |
| `provision` | Combination of install, upgrade, lint, and template for Helm |
| `rollback` | PRoll back a release to a previous revision           |
| `status` | Status of a Helm release  |
| `template` |  Render chart templates           |
| `uninstall` | Uninstall a Helm release  |
| `upgrade` | Upgrade a deployed Helm chart  |
| `repo add` | Add a chart repository |
| `repo update` | Update chart repositories |
| `pull` | Download a chart from a repository |
| `plugin install` | Install one or more Helm plugins (comma-separated). |
| `history` | Show revision history for a release |
