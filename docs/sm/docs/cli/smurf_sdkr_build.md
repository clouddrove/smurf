## smurf sdkr build

Build a Docker image with the given name and tag.

```
smurf sdkr build [IMAGE[:TAG]] [flags]
```

### Examples

```

smurf sdkr build my-image:v1
smurf sdkr build my-image:v1 --file Dockerfile --context ./build-context --no-cache --build-arg key1=value1 --build-arg key2=value2 --target my-target --platform linux/amd64 --timeout 400
smurf sdkr build
# In the last example, it will read "image:v1" from config and use the parsed image name and tag

```

### Options

```
      --ai                      To enable AI help mode, export the OPENAI_API_KEY environment variable with your OpenAI API key.
      --build-arg stringArray   Set build-time variables
      --buildkit                Enable BuildKit for advanced Dockerfile features
      --context string          Build context directory (default: current directory)
  -f, --file string             Path to Dockerfile relative to context directory
  -h, --help                    help for build
      --no-cache                Do not use cache when building the image
      --platform string         Set the platform for the build (e.g., linux/amd64, linux/arm64)
      --target string           Set the target build stage to build
      --timeout int             Set the build timeout in seconds (default 1500)
```

### SEE ALSO

* [smurf sdkr](smurf_sdkr.md)	 - Subcommand for Docker-related actions

