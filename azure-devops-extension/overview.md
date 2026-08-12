# Smurf Azure Pipelines

Install and run the Smurf CLI in Azure Pipelines.

Smurf wraps common Terraform, Docker, and Helm workflows behind one CLI, so the
same commands can run locally, in GitHub Actions, and in Azure Pipelines.

## Usage

```yaml
steps:
  - task: Smurf@1
    displayName: Install Smurf
    inputs:
      version: v1.1.5
      command: ""

  - script: smurf version
    displayName: Smurf version

  - script: smurf stf plan
    displayName: Terraform plan
```

You can also run a single Smurf command directly in the task:

```yaml
steps:
  - task: Smurf@1
    inputs:
      version: latest
      command: stf plan
```

## Inputs

| Input | Default | Description |
| --- | --- | --- |
| `version` | `latest` | Smurf version to install. Use `latest` or a release tag such as `v1.1.5`. |
| `command` | `version` | Smurf command and arguments to run. Leave blank to install only. |
| `workingDirectory` | `$(System.DefaultWorkingDirectory)` | Directory where the Smurf command runs. |

## Links

- Repository: https://github.com/clouddrove/smurf
- Documentation: https://smurf.clouddrove.com
- Issues: https://github.com/clouddrove/smurf/issues
