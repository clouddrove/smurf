# Docker User Guide 🐳

Use `smurf sdkr <command>` to run smurf sdkr commands. Supported commands include:

- **`build`**: Builds a Docker image with the specified **name** and **tag**.  
- **`provision-acr`**: Builds and pushes a Docker image to **Azure Container Registry (ACR)**.  
- **`provision-ecr`**: Builds and pushes a Docker image to **AWS Elastic Container Registry (ECR)**.  
- **`provision-gcr`**: Builds and pushes a Docker image to **Google Container Registry (GCR)**.  
- **`provision-hub`**: Builds, scans, and pushes a Docker image to **Docker Hub** for enhanced security.
- **`provision-ghcr`**: Builds, scans, and pushes a Docker image to **GitHub Container Registry**
- **`push`**: Pushes Docker images to **ACR, ECR, GCR,** or **Docker Hub** in one simple command.  
- **`remove`**: Deletes a Docker image from your **local system** to free up space.  
- **`scan`**: Analyzes a Docker image for known **security vulnerabilities** before deployment.  
- **`tag`**: Tags a Docker image for easy **identification** and **repository management**.   

## Using Smurf Docker in local environment
Suppose you want to build and push a docker image to AWS Elastic Container Registry (ECR).To do this run the command: 
```bash
smurf sdkr <ecr_url>
```
![sdkr](gif/sdkr_ecr.mov)

Suppose you want to scan a docker image named smurf using smurf. 
To do this run the command: smurf sdkr scan <img_name>

```bash
smurf sdkr scan <img_name>
```
![sdkr](gif/sdkr_scan.mov)

## Using Smurf Docker in GitHub Actions
Using Smurf Docker in GitHub Actions involves calling the Smurf shared workflow.
To Build and Push Image to AWS ECR workflow will look like-
```yaml
jobs:
  build-and-push:
    runs-on: ubuntu-latest
    env:
      AWS_ACCESS_KEY_ID: ${{ secrets.AWS_ACCESS_KEY_ID }}
      AWS_SECRET_ACCESS_KEY: ${{ secrets.AWS_SECRET_ACCESS_KEY }}
      AWS_SESSION_TOKEN: ${{ secrets.AWS_SESSION_TOKEN }}
      aws-region: us-east-1

    steps:
      - name: Set up smurf
        uses: clouddrove/smurf@master
        with:
          version: latest

      - name: Build and push Docker image
        run: |
          smurf sdkr provision-ecr <ecr_url>
```

## Using smurf.yaml configure file for Smurf Docker
Use the smurf.yaml configuration file to perform Smurf Docker both locally and in GitHub Actions.
```bash
smurf sdkr init
```
it create the `smurf.yaml` configure for docker 
```yaml
sdkr:
  imageName: "my-application"
  targetImageTag: "v1.0.0"
```

Once the `smurf.yaml` file is configured, the workflow to build and push an image to AWS ECR will look like this:
```yaml
jobs:
  build-and-push:
    runs-on: ubuntu-latest
    env:
      AWS_ACCESS_KEY_ID: ${{ secrets.AWS_ACCESS_KEY_ID }}
      AWS_SECRET_ACCESS_KEY: ${{ secrets.AWS_SECRET_ACCESS_KEY }}
      AWS_SESSION_TOKEN: ${{ secrets.AWS_SESSION_TOKEN }}
      aws-region: us-east-1

    steps:
      - name: Set up smurf
        uses: clouddrove/smurf@master
        with:
          version: latest

      - name: Build and push Docker image
        run: |
          smurf sdkr provision-ecr
```

![sdkr](gif/sdkr_provision_ecr.mov)