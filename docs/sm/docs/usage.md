# Getting Started

## Using in Local CLI

### Example for Smurf SELM

Use `smurf selm <command>` to run Helm commands. Supported commands include:

- **Help:** `smurf selm --help`
- **Create a Helm Chart:** `smurf selm create`
- **Install a Chart:** `smurf selm install`
- **Upgrade a Release:** `smurf selm upgrade`
- **Provision Helm Environment:** `smurf selm provision --help`

The `provision` command for Helm combines `install`, `upgrade`, `lint`, and `template`.

## Using in GitHub Actions

### Example for scan and push docker image on ECR

```yaml
name: Smurf sdkr provision-ecr

on:
  push:
    branches:
      - master
  workflow_dispatch:

jobs:
  docker-build-publish:
    runs-on: ubuntu-latest
    permissions: write-all
    
    env:
      DOCKER_USERNAME: ${{ secrets.DOCKERHUB_USERNAME }}
      DOCKER_PASSWORD: ${{ secrets.DOCKERHUB_TOKEN }}
     
    steps:
      - name: Checkout code
        uses: actions/checkout@v4.1.7

      - name: Configure AWS credentials
        uses: aws-actions/configure-aws-credentials@v4
        with:
          role-to-assume: ${{ secrets.AWS_GITHUB_OIDC_ROLE }}
          role-session-name: aws-auth
          aws-region: 

      - name: Login to Amazon ECR
        id: login-ecr
        uses: aws-actions/amazon-ecr-login@v1
 
      - name: Run Provision-ecr (Build,Scan and Push)
        uses: clouddrove/smurf@v1.0.0
        with:
          tool: sdkr
          command: provision-ecr repo:image_name -f Dockerfile --yes
```