### Using Smurf SDKR on local
Use `smurf sdkr <command> <flags>` to run Docker commands. Supported commands include:

- **Help:** `smurf sdkr --help`
- **Build an Image:** `smurf sdkr build`
- **Scan an Image:** `smurf sdkr scan`
- **Push an Image:** `smurf sdkr push --help`
- **Provision Registry Environment:** `smurf sdkr provision-hub [flags] `(for Docker Hub)

The `provision-hub` command for Docker combines `build` and `push`.
The `provision-ecr` command for Docker combines `build` and `push` for AWS ECR.
The `provision-gcp` command for Docker combines `build` and `push` for GCP (GCR or Artifact Registry).
The `provision-acr` command for Docker combines `build` and `push` for Azure ACR.
The `provision-ghcr` command for Docker combines `build` and `push` for GitHub Container Registry.

Each `provision-*` command prompts `Proceed with push? [y/N]` before pushing when run on a TTY; pass `--yes` to skip the prompt (e.g. in CI).

### Using Smurf SDKR in GitHub Action
### This GitHub Action builds docker image and pushes to the registry you want.

```yaml
name: Smurf SDKR Workflow

on:
  push:
    branches: main

jobs:
  login:
    runs-on: ubuntu-latest
    steps:
     - name: Checkout repository
       uses: actions/checkout@v3
      
      - name: Setup Smurf
        uses: clouddrove/smurf@v1.1.2

      - name: Smurf sdkr build
        run: |
          smurf sdkr build image_name:tag -f Dockerfile

      - name: Smurf sdkr push image
        run: |
          smurf sdkr push hub USERNAME/image_name:tag
```

### All available commands in Smurf SDKR

| Command   | Description                          |
|-----------|--------------------------------------|
| `build`    | Build a Docker image with the given name and tag |
| `init` | Create a default smurf.yaml file with sdkr configuration |
| `provision-acr` | Build and push a Docker image to Azure Container Registry          |
| `provision-ecr`    | Build and push a Docker image to AWS ECR  |
| `provision-gcp`   | Build and push a Docker image to Google Container Registry or Artifact Registry               |
| `provision-ghcr` | Build and push a Docker image to GitHub Container Registry |
| `provision-hub` | Build and push a Docker image to Docker Hub            |
| `push` | Push cmd helps to push images to Docker Hub, ACR, GCR, ECR           |
| `push az` | Push a Docker image to Azure Container Registry |
| `push aws` | Push Docker images to ECR |
| `push gcp` | Push Docker images to Google Container Registry or Artifact Registry |
| `push hub` | Push Docker images to Docker Hub |
| `remove` | Remove a Docker image from the local system           |
| `scan` | Scan a Docker image for known vulnerabilities           |
| `tag` | Tag a Docker image for a remote repository  |