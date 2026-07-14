## smurf sdkr provision-hub

Build and push a Docker image.

### Synopsis

Build and push a Docker image to Docker Hub.
	Set DOCKER_USERNAME and DOCKER_PASSWORD environment variables for Docker Hub authentication, for example:
  	export DOCKER_USERNAME="your-username"
  	export DOCKER_PASSWORD="your-password"

```
smurf sdkr provision-hub [IMAGE_NAME[:TAG]] [flags]
```

### Examples

```

  # Provide "myuser/myimage:latest" as an argument
  smurf sdkr provision-hub myuser/myimage:latest --context . --file Dockerfile --no-cache \
    --build-arg key1=value1 --build-arg key2=value2 --target my-target --platform linux/amd64 \
    --yes --delete

  # If you omit the argument, it will read from config and rely on "image_name" from there
  smurf sdkr provision-hub --yes --delete

```

### Options

```
      --ai                      To enable AI help mode, export the OPENAI_API_KEY environment variable with your OpenAI API key.
      --build-arg stringArray   Set build-time variables (e.g. --build-arg key=value)
      --context string          Build context directory (default: current directory)
  -d, --delete                  Delete the local image after pushing
  -f, --file string             Dockerfile path relative to the context directory (default: 'Dockerfile')
  -h, --help                    help for provision-hub
      --no-cache                Do not use cache when building the image
      --platform string         Set the platform for the image (e.g., linux/amd64)
      --target string           Set the target build stage to build
      --timeout int             Build timeout (default 1500)
  -y, --yes                     Push the image without confirmation
```

### SEE ALSO

* [smurf sdkr](smurf_sdkr.md)	 - Subcommand for Docker-related actions

