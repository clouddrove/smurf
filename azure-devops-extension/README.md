# Smurf Azure DevOps Task

This folder contains the Azure DevOps Marketplace extension for the `Smurf@1`
pipeline task.

## Usage

```yaml
steps:
  - task: Smurf@1
    displayName: Run smurf
    inputs:
      version: latest
      command: stf plan
      workingDirectory: $(System.DefaultWorkingDirectory)
```

To install only and make `smurf` available to later steps:

```yaml
steps:
  - task: Smurf@1
    inputs:
      version: v1.1.5
      command: ""

  - script: smurf version
    displayName: Verify smurf
```

## Development

Install task dependencies:

```bash
cd azure-devops/Smurf
npm install
```

## Codex release skill

This repository includes a Codex skill for publishing extension updates:

```bash
export AZURE_DEVOPS_MARKETPLACE_PAT='<PAT>'
.codex/skills/azure-devops-extension-publisher/scripts/publish-extension.sh
```

The script auto-bumps the patch version, validates the task, creates the VSIX,
and publishes the public Marketplace extension. Pass `--version X.Y.Z` to
publish a specific version.

## GitHub Actions release workflow

The root repository workflow `.github/workflows/publish-azure-devops-extension.yml`
publishes the extension from CI.

Required repository secret:

```text
AZURE_DEVOPS_MARKETPLACE_PAT
```

Run the workflow manually and optionally provide a version such as `1.0.2`.
If no version is provided, the workflow bumps the patch version automatically.

Package the extension from `azure-devops/`:

```bash
npm install -g tfx-cli
cd azure-devops
tfx extension create --manifest-globs vss-extension.json
```

Before publishing, update these versions:

- `azure-devops/vss-extension.json`
- `azure-devops/Smurf/task.json`
- `azure-devops/Smurf/package.json`

Publish privately first and share it with a test Azure DevOps organization.
