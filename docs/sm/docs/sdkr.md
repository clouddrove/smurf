# Docker User Guide

### Using Smurf SDKR on local

Use `smurf sdkr <command> <flags>` to run smurf sdkr commands. Supported commands include:

- **Help:** `smurf sdkr --help`
- **Build an Image:** `smurf sdkr build`
- **Scan an Image:** `smurf sdkr scan`
- **Push an Image:** `smurf sdkr push --help`
- **Provision Registry Environment:** `smurf sdkr provision-hub [flags] `(for Docker Hub)

### Docker Provision Commands

- The `provision-ecr` command for Docker combines `build`, `scan`, and `publish` for AWS ECR.  

- The `provision-hub` command for Docker combines `build`, `scan`, and `publish`.  

- The `provision-gcr` command for Docker combines `build`, `scan`, and `publish` for GCP GCR.  

- The `provision-acr` command for Docker combines `build`, `scan`, and `publish` for Azure ACR.  


#### Usage
The following workflow can build,scan and push a Docker image locally, providing vulnerability results under the code scanning section of the security tab. It also allows you to choose which vulnerability should block the workflow before pushing the Docker image to the Docker registry this workflow support DOCKERHUB, ECR or both.

### Example for scan and push docker image on Dockerhub

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
        uses: clouddrove-sandbox/smurf-custon-github-action-test@master
        with:
          tool: sdkr
          command: build image_name:tag

      - name: Smurf sdkr push image
        uses: clouddrove-sandbox/smurf-custon-github-action-test@master
        with: 
          tool: sdkr
          command: push hub USERNAME/image_name:tag
```

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
        uses: clouddrove-sandbox/smurf-custon-github-action-test@master
        with:
          tool: sdkr
          command: provision-ecr repo:image_name -f Dockerfile --yes
```

## Sdkr Build Flags

Below is a list of available flags for the `docker build` command:

### Available Flags

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


