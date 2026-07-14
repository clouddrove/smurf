## smurf sdkr provision-ecr

Build and push a Docker image to AWS ECR.

```
smurf sdkr provision-ecr [IMAGE_NAME[:TAG]] [flags]
```

### Examples

```

  # IMAGE_NAME can be in the form:
  #   123456789012.dkr.ecr.us-east-1.amazonaws.com/my-repo:python
  smurf sdkr provision-ecr 123456789012.dkr.ecr.us-east-1.amazonaws.com/my-repo:python \
      --no-cache \
      --build-arg key1=value1 \
      --build-arg key2=value2 \
      --target my-target \
      --platform linux/amd64 \
      --yes \
      --delete

```

### Options

```
      --ai                      To enable AI help mode, export the OPENAI_API_KEY environment variable with your OpenAI API key.
  -a, --build-arg stringArray   Set build-time variables
      --context string          Build context directory (default: current directory)
  -d, --delete                  Delete the local image after pushing
  -f, --file string             Dockerfile path relative to context directory (default: 'Dockerfile')
  -h, --help                    help for provision-ecr
  -c, --no-cache                Do not use cache when building the image
  -p, --platform string         Platform for the image
  -t, --target string           Set the target build stage to build
      --timeout int             Build timeout (default 1500)
  -y, --yes                     Push the image to ECR without confirmation
```

### SEE ALSO

* [smurf sdkr](smurf_sdkr.md)	 - Subcommand for Docker-related actions

