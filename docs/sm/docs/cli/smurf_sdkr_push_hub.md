## smurf sdkr push hub

Push Docker images to Docker Hub

### Synopsis


Push Docker images to Docker Hub.
Export DOCKER_USERNAME and DOCKER_PASSWORD as environment variables for Docker Hub authentication, for example:
  export DOCKER_USERNAME="your-username"
  export DOCKER_PASSWORD="your-password"

```
smurf sdkr push hub [IMAGE_NAME[:TAG]] [flags]
```

### Examples

```

  smurf sdkr push hub myapp:v1
  smurf sdkr push hub myapp:v1 --delete

```

### Options

```
      --ai            To enable AI help mode, export the OPENAI_API_KEY environment variable with your OpenAI API key.
  -d, --delete        Delete the local image after pushing
  -h, --help          help for hub
      --timeout int   Timeout for the push operation in seconds (default 1500)
```

### SEE ALSO

* [smurf sdkr push](smurf_sdkr_push.md)	 - Push cmd helps to push images to Docker Hub, ACR, GCR, ECR

