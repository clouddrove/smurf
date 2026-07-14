## smurf sdkr provision-acr

Build and push a Docker image to Azure Container Registry.

```
smurf sdkr provision-acr [IMAGE_NAME[:TAG]] [flags]
```

### Examples

```

  smurf sdkr provision-acr myimage:v1 -s <SUBSCRIPTION_ID> -r <RESOURCE_GROUP> -g <REGISTRY_NAME>
  smurf sdkr provision-acr -f Dockerfile -c -a key1=value1 -a key2=value2 -t my-target -p linux/amd64 -y -d

```

### Options

```
      --ai                       To enable AI help mode, export the OPENAI_API_KEY environment variable with your OpenAI API key.
  -a, --build-arg stringArray    Set build-time variables
      --context string           Build context directory (default: current directory)
  -d, --delete                   Delete the local image after pushing
  -f, --file string              path to Dockerfile relative to context directory
  -h, --help                     help for provision-acr
  -c, --no-cache                 Do not use cache when building the image
  -p, --platform string          Platform for the image
  -g, --registry-name string     Azure Container Registry name (required)
  -r, --resource-group string    Azure resource group name (required)
  -s, --subscription-id string   Azure subscription ID (required)
  -t, --target string            Set the target build stage to build
      --timeout int              Build timeout (default 1500)
  -y, --yes                      Push the image to ACR without confirmation
```

### SEE ALSO

* [smurf sdkr](smurf_sdkr.md)	 - Subcommand for Docker-related actions

