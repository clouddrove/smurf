## smurf sdkr provision-ghcr

Build and push a Docker image to GitHub Container Registry

### Synopsis

Build and push a Docker image to GitHub Container Registry (GHCR).

Authentication:
  - Set GITHUB_USERNAME and GITHUB_TOKEN environment variables
  - OR define them in config file (GITHUB_USERNAME, GITHUB_TOKEN)
  - The token must have 'write:packages' scope

Image format:
  ghcr.io/OWNER/IMAGE_NAME:TAG
Example: ghcr.io/my-org/my-app:latest

```
smurf sdkr provision-ghcr [IMAGE_NAME[:TAG]] [flags]
```

### Examples

```

  # Push to GHCR with full image reference
  smurf sdkr provision-ghcr ghcr.io/my-org/my-image:latest

  # Push with specific tag and build options
  smurf sdkr provision-ghcr ghcr.io/my-username/my-app:v1.0.0 \
    --context . --file Dockerfile --no-cache \
    --build-arg ENV=production --platform linux/amd64 \
    --delete

  # Using environment variables for auth
  export GITHUB_USERNAME="my-username"
  export GITHUB_TOKEN="ghp_yourPersonalAccessToken"
  smurf sdkr provision-ghcr ghcr.io/my-org/my-app:latest

  # Read image name from config file
  smurf sdkr provision-ghcr --delete

```

### Options

```
      --ai                      To enable AI help mode, export the OPENAI_API_KEY environment variable with your OpenAI API key.
      --build-arg stringArray   Set build-time variables (key=value). Repeat the flag or pass comma-separated pairs
      --context string          Build context (default: current directory)
  -d, --delete                  Delete local image after push
  -f, --file string             Path to Dockerfile (default: Dockerfile)
  -h, --help                    help for provision-ghcr
      --no-cache                Disable build cache
      --platform string         Platform (e.g. linux/amd64)
      --target string           Target build stage
      --timeout int             Build timeout in seconds (default 1500)
  -y, --yes                     Push without confirmation
```

### SEE ALSO

* [smurf sdkr](smurf_sdkr.md)	 - Subcommand for Docker-related actions

