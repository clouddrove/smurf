## smurf sdkr push gcp

Push Docker images to Google Container Registry or Artifact Registry

### Synopsis

Push Docker images to Google Container Registry or Artifact Registry. 

Authentication Methods:
1. gcloud CLI (recommended): Run 'gcloud auth login' and 'gcloud auth configure-docker'
2. Service Account: Set GOOGLE_APPLICATION_CREDENTIALS environment variable

Supports:
- GCR: gcr.io/PROJECT_ID/IMAGE_NAME:TAG
- Artifact Registry: REGION-docker.pkg.dev/PROJECT_ID/REPOSITORY/IMAGE_NAME:TAG
- Short form: IMAGE_NAME:TAG (automatically uses Artifact Registry)


```
smurf sdkr push gcp [IMAGE_NAME[:TAG]] [flags]
```

### Examples

```
  # Push to Artifact Registry with full image name
  smurf sdkr push gcp us-central1-docker.pkg.dev/my-project/my-repo/myapp:v1

  # Push to GCR with full image name
  smurf sdkr push gcp gcr.io/my-project/myapp:v1

  # Push with short name (uses Artifact Registry)
  smurf sdkr push gcp myapp:v1 --project-id my-project

  # Push and delete local image
  smurf sdkr push gcp myapp:v1 --project-id my-project --delete
```

### Options

```
      --ai                  To enable AI help mode, export the OPENAI_API_KEY environment variable with your OpenAI API key.
  -d, --delete              Delete the local image after pushing
  -h, --help                help for gcp
      --project-id string   GCP project ID (required for short image names)
```

### SEE ALSO

* [smurf sdkr push](smurf_sdkr_push.md)	 - Push cmd helps to push images to Docker Hub, ACR, GCR, ECR

