## Docker User Guide

Using Smurf SDKR on local

Use `smurf sdkr <command> <flags>` to run smurf sdkr commands. Supported commands include:

- **Help:** `smurf sdkr --help`
- **Build an Image:** `smurf sdkr build`
- **Scan an Image:** `smurf sdkr scan`
- **Push an Image:** `smurf sdkr push --help`
- **Provision Registry Environment:** `smurf sdkr provision-hub [flags] `(for Docker Hub)

Docker Provision Commands

- The `provision-ecr` command for Docker combines `build`, `scan`, and `publish` for AWS ECR.  

- The `provision-hub` command for Docker combines `build`, `scan`, and `publish`.  

- The `provision-gcr` command for Docker combines `build`, `scan`, and `publish` for GCP GCR.  

- The `provision-acr` command for Docker combines `build`, `scan`, and `publish` for Azure ACR.  


Usage
The following workflow can build,scan and push a Docker image locally, providing vulnerability results under the code scanning section of the security tab. It also allows you to choose which vulnerability should block the workflow before pushing the Docker image to the Docker registry this workflow support DOCKERHUB, ECR or both.

Example for scan and push docker image on Dockerhub

```yaml
name: Smurf Docker Build and Publish

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
          aws-region: #AWS_REGION


      - name: Smurf sdkr build
        uses: clouddrove/smurf@v1.0.0
        with:
          tool: sdkr
          command: build image_name:tag

      - name: Smurf sdkr push image
        uses: clouddrove/smurf@v1.0.0
        with: 
          tool: sdkr
          command: push hub USERNAME/image_name:tag
```

Example for scan and push docker image on ECR

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

Sdkr Build Flags

Below is a list of available flags for the `docker build` command:

Available Flags

| Flag                      | Type          | Description                                          | Default Value            |
|---------------------------|--------------|------------------------------------------------------|--------------------------|
| `--build-arg`            | `stringArray` | Set build-time variables                            | N/A                      |
| `--context`              | `string`      | Build context directory                            | Current directory        |
| `-f`, `--file`           | `string`      | Path to Dockerfile relative to context directory   | N/A                      |
| `-h`, `--help`           | N/A           | Display help for the build command                 | N/A                      |
| `--no-cache`             | N/A           | Do not use cache when building the image           | N/A                      |
| `--platform`             | `string`      | Set the platform for the build (e.g., linux/amd64, linux/arm64) | N/A |
| `--target`               | `string`      | Set the target build stage to build                | N/A                      |
| `--timeout`              | `int`         | Set the build timeout                              | `1500`                   |


## Helm User Guide

Using Smurf SELM on local

Use `smurf selm <command>` to run Helm commands. Supported commands include:

- **Help:** `smurf selm --help`
- **Create a Helm Chart:** `smurf selm create`
- **Install a Chart:** `smurf selm install`
- **Upgrade a Release:** `smurf selm upgrade`
- **Provision Helm Environment:** `smurf selm provision --help`

The `provision` command for Helm combines `install`, `upgrade`, `lint`, and `template`.

Context:
This workflow is used to upgrade the Helm charts using GitHub Actions. It utilizes the workflows defined in `.github/workflows/selm.yml`

Usage
The selm workflow can be triggered manually using the GitHub Actions workflow dispatch feature. It deploys or rollback Helm charts based on the specified inputs. Additionally, it also performs Helm template and Helm lint operations.
To use the selm Workflow, add the following workflow definition to your `.github/workflows/selm.yml` file:

Example for AWS cloud provider

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
        uses: clouddrove/smurf@v1.0.0
        with:
          tool: sdkr
          command: provision-ecr repo:image_name

      - name: Smurf SELM Upgrade
        uses: clouddrove/smurf@v1.0.0
        with:
          tool: selm
          command: upgrade release_name --install --atomic --set image.tag=${{ env.tag }} -f values.yaml ./my_chart --namespace  --timeout int

```

Selm Deployment Parameters

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

## Terraform User Guide


Using Smurf STF on local
Use `smurf stf <command>` to run Terraform commands. Supported commands include:

- **Help:** `smurf stf --help`
- **Initialize Terraform:** `smurf stf init`
- **Generate and Show Execution Plan:** `smurf stf plan`
- **Apply Terraform Changes:** `smurf stf apply`
- **Detect Drift in Terraform State:** `smurf stf drift`
- **Provision Terraform Environment:** `smurf stf provision`

The `provision` command for Terraform performs `init`, `validate`, and `apply`.

Using Smurf STF in GitHub Action
 This GitHub Action Initialize Terraform and Validate Terraform changes.

```yaml
name: Smurf STF Workflow

on:
  push:
    branches:
      - master

jobs:
  terraform:
    runs-on: ubuntu-latest
    steps:
      - name: Checkout repository
        uses: actions/checkout@v3

      - name: Smurf stf init
        uses: clouddrove/smurf@v1.0.0
        with:
          path:   # we can specify the path of folder where main.tf is located.
          tool: stf
          command: init

      - name: Smurf stf validate
        uses: clouddrove/smurf@v1.0.0
        with:
          path:   
          tool: stf
          command: validate
```
All available commands in Smurf STF
| Command   | Description                          |
|-----------|--------------------------------------|
| `apply`    | Apply the changes required to reach the desired state of Terraform Infrastructure |
| `destroy` | Destroy the Terraform Infrastructure |
| `drift`    | Detect drift between state and infrastructure  for Terraform  |
| `format`   | Format the Terraform Infrastructure              |
| `init` | Initialize Terraform             |
| `output` | Generate output for the current state of Terraform Infrastructure  |
| `plan` | Generate and show an execution plan for Terraform          |
| `provision` | Its the combination of init, plan, apply, output for Terraform |
| `validate` | Validate  Terraform changes |


Available Flags for Provision Command

| Flag                  | Description                                                    | Default Value |
|-----------------------|----------------------------------------------------------------|--------------|
| `--approve`          | Skip interactive approval of plan before applying             | `true`       |
| `-h, --help`         | Display help for the provision command                        | N/A          |
| `--var string`       | Specify a variable in 'NAME=VALUE' format                     | N/A          |
| `--var-file string`  | Specify a file containing variables                           | N/A          |
