# Helm User Guide

### Using Smurf SELM on local

Use `smurf selm <command>` to run Helm commands. Supported commands include:

- **Help:** `smurf selm --help`
- **Create a Helm Chart:** `smurf selm create`
- **Install a Chart:** `smurf selm install`
- **Upgrade a Release:** `smurf selm upgrade`
- **Provision Helm Environment:** `smurf selm provision --help`

- The `provision` command for Helm combines `install`, `upgrade`, `lint`, and `template`.

### Context:
This workflow is used to upgrade the Helm charts using GitHub Actions. It utilizes the workflows defined in `.github/workflows/selm.yml`

#### Usage
The selm workflow can be triggered manually using the GitHub Actions workflow dispatch feature. It deploys or rollback Helm charts based on the specified inputs. Additionally, it also performs Helm template and Helm lint operations.
To use the selm Workflow, add the following workflow definition to your `.github/workflows/selm.yml` file:

#### Example for AWS cloud provider

```yaml
name: Smurf selm

on:
  push:
    branches:
      - master
  workflow_dispatch:

jobs:
  smurf-selm:
    runs-on: ubuntu-latest
    permissions: write-all
    env:
  
    steps:
      - name: Check out in repo
        uses: actions/checkout@v2.3.4

      - name: Configure AWS credentials with OIDC
        uses: aws-actions/configure-aws-credentials@v4
        with:
          role-to-assume: #oidc role for aws account authentication
          aws-region: #aws_region

      - name: Set environment variables
        run: |
          echo "AWS_DEFAULT_REGION= " >> $GITHUB_ENV  #add the default region
          echo "EKS_CLUSTER_NAME= " >> $GITHUB_ENV    #add the eks_cluster name
      
      - name: Smurf SDKR Provision (for image building and ecr push)
        uses: clouddrove-sandbox/smurf-custon-github-action-test@master
        with:
          tool: sdkr
          command: provision-ecr repo:image_name

      - name: Smurf SELM Upgrade
        uses: clouddrove-sandbox/smurf-custon-github-action-test@master
        with:
          tool: selm
          command: upgrade release_name --install --atomic --set image.tag=${{ env.tag }} -f values.yaml ./my_chart --namespace  --timeout int

```

## Selm Deployment Parameters

| Parameter       | Description | Required |
|---------------|-------------|----------|
| **release** | Helm release name. Will be combined with track if set. | ✅ Yes |
| **namespace** | Kubernetes namespace name. | ✅ Yes |
| **chart** | Helm chart path. If set to `"app"`, this will use the built-in Helm chart found in this repository. | ✅ Yes |
| **values** | Helm chart values, expected to be a YAML or JSON string. | ❌ No |
| **token** | GitHub repository token. If included and the event is a deployment, then the `deployment_status` event will be fired. | ❌ No |
| **version** | Version of the app, usually the commit SHA works here. | ❌ No |
| **timeout** | Specify a timeout for Helm deployment. | ❌ No |
| **repo** | Helm chart repository to be added. | ❌ No |
| **repo-username** | Helm repository username if authentication is needed. | ❌ No |
| **repo-password** | Helm repository password if authentication is needed. | ❌ No |
| **atomic** | If `true`, the upgrade process rolls back changes made in case of a failed upgrade. Defaults to `true`. | ❌ No |

