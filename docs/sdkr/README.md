### Using Smurf SDKR on local
Use `smurf sdkr <command> <flags>` to run Docker commands. Supported commands include:

- **Help:** `smurf sdkr --help`
- **Build an Image:** `smurf sdkr build`
- **Scan an Image:** `smurf sdkr scan`
- **Push an Image:** `smurf sdkr push --help`
- **Provision Registry Environment:** `smurf sdkr provision-hub [flags] `(for Docker Hub)

The `provision-hub` command for Docker combines `build`, `scan`, and `publish`.
The `provision-ecr` command for Docker combines `build`, `scan`, and `publish` for AWS ECR.
The `provision-gcr` command for Docker combines `build`, `scan`, and `publish` for GCP GCR.
The `provision-acr` command for Docker combines `build`, `scan`, and `publish` for Azure ACR.

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
        uses: clouddrove/smurf@v0.0.4

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
| `provision-acr` | Build, scan and push a Docker image to Azure Container Registry          |
| `provision-ecr`    | Build, scan, and push a Docker image to AWS ECR  |
| `provision-gcr`   | Build, scan, and push a Docker image to Google Container Registry                |
| `provision-hub` | Build, scan, and push a Docker image to Docker Hub             |
| `push` | Push cmd helps to push images to Docker Hub, ACR, GCR, ECR           |
| `remove` | Remove a Docker image from the local system           |
| `scan` | Scan a Docker image for known vulnerabilities           |
| `tag` | Tag a Docker image for a remote repository  |