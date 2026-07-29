---
name: azure-devops-extension-publisher
description: Update, package, validate, and publish this Smurf Azure DevOps Marketplace extension. Use when asked to release or update the Azure DevOps extension from this repository; the only required secret is a Marketplace PAT supplied at runtime.
---

# Azure DevOps Extension Publisher

Use this skill from the `azure-devops-extension/` repository root.

## What This Skill Does

Publishes the Smurf Azure DevOps Marketplace extension by:

1. Bumping `vss-extension.json`, `Smurf/task.json`, and `Smurf/package.json`.
2. Refreshing `Smurf/package-lock.json`.
3. Installing task runtime dependencies.
4. Running JSON, JavaScript, and npm audit checks.
5. Creating the VSIX with `tfx extension create`.
6. Publishing with `tfx extension publish --no-wait-validation`.

## Inputs

Required:

- Marketplace PAT with permission to publish extensions.

Optional:

- Explicit version, for example `1.0.2`. If omitted, the script bumps the current patch version.

## Standard Workflow

Never print or persist the PAT. Prefer an environment variable:

```bash
export AZURE_DEVOPS_MARKETPLACE_PAT='<PAT>'
.codex/skills/azure-devops-extension-publisher/scripts/publish-extension.sh
```

To publish a specific version:

```bash
export AZURE_DEVOPS_MARKETPLACE_PAT='<PAT>'
.codex/skills/azure-devops-extension-publisher/scripts/publish-extension.sh --version 1.0.2
```

If the user insists on passing the PAT directly:

```bash
.codex/skills/azure-devops-extension-publisher/scripts/publish-extension.sh --pat '<PAT>'
```

## Validation

After publishing, check Marketplace validation:

```bash
tfx extension isvalid \
  --publisher clouddrove \
  --extension-id smurf-azure-pipelines \
  --version <VERSION> \
  --service-url https://marketplace.visualstudio.com/ \
  --token '<PAT>'
```

The public listing is:

```text
https://marketplace.visualstudio.com/items?itemName=clouddrove.smurf-azure-pipelines
```

## Guardrails

- Do not reuse an existing published version; Marketplace requires every publish to increase the extension version.
- Keep `vss-extension.json` and `Smurf/task.json` major versions aligned. `1.x.y` publishes `Smurf@1`.
- Do not use `--share-with` for public releases.
- If Marketplace rejects the package, fix the validation error and publish a newer version.
- Treat any PAT pasted into chat, logs, or commits as compromised and tell the user to revoke it.
