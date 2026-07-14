## smurf sdkr provision-gcp

Build and push a Docker image to Google Container Registry or Artifact Registry.

### Synopsis

Build and push a Docker image to Google Container Registry or Artifact Registry.
Set the GOOGLE_APPLICATION_CREDENTIALS environment variable to the path of your service account JSON key file.

Supports:
- Full Artifact Registry path: us-central1-docker.pkg.dev/PROJECT/REPO/IMAGE:TAG
- Full GCR path: gcr.io/PROJECT/IMAGE:TAG  
- Short form: IMAGE:TAG (automatically uses Artifact Registry)
- Repository form: REPO/IMAGE:TAG (automatically uses Artifact Registry)


```
smurf sdkr provision-gcp [IMAGE_NAME[:TAG]] [flags]
```

### Examples

```

  # Build and push using full Artifact Registry path
  smurf sdkr provision-gcp us-central1-docker.pkg.dev/my-project/smurf-test/smurfimage/smurfab:v1

  # Build and push using full GCR path
  smurf sdkr provision-gcp gcr.io/my-project/myapp:v1.0

  # Build and push with short name (auto Artifact Registry)
  smurf sdkr provision-gcp myapp:v1.0 --project-id my-project

  # Build and push with repository path (auto Artifact Registry)
  smurf sdkr provision-gcp my-repo/myapp:v1.0 --project-id my-project

  # Build and push with short name to GCR
  smurf sdkr provision-gcp myapp:v1.0 --project-id my-project --use-gcr

  # With additional options
  smurf sdkr provision-gcp myapp:v1.0 --project-id my-project --file Dockerfile --no-cache \
    --build-arg key1=value1,key2=value2 --target my-target \
    --delete --platform linux/amd64

```

### Options

```
      --ai                      To enable AI help mode, export the OPENAI_API_KEY environment variable with your OpenAI API key.
  -a, --build-arg stringArray   Set build-time variables (key=value). Repeat the flag or pass comma-separated pairs
      --context string          Build context directory (default: current directory)
  -d, --delete                  Delete the local image after pushing
  -f, --file string             Name of the Dockerfile relative to the context directory (default: 'Dockerfile')
  -h, --help                    help for provision-gcp
  -c, --no-cache                Do not use cache when building the image
  -p, --platform string         Set the platform for the image (e.g., linux/amd64)
      --project-id string       GCP project ID (required for short image names)
  -t, --target string           Set the target build stage to build
      --timeout int             Build timeout in seconds (default 1500)
      --use-gcr                 Use legacy Google Container Registry (gcr.io) instead of Artifact Registry
  -y, --yes                     Push the image to registry without confirmation
```

### SEE ALSO

* [smurf sdkr](smurf_sdkr.md)	 - Subcommand for Docker-related actions

