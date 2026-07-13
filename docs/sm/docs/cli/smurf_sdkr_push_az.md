## smurf sdkr push az

Push a Docker image to Azure Container Registry.

```
smurf sdkr push az [IMAGE_NAME[:TAG]] [flags]
```

### Examples

```

  smurf sdkr push az myapp:v1 -s <subscription-id> -r <resource-group> -g <registry-name> --delete
  smurf sdkr push az myapp:v1 --subscription-id <subscription-id> --resource-group <resource-group> --registry-name <registry-name>
  
```

### Options

```
      --ai                       To enable AI help mode, export the OPENAI_API_KEY environment variable with your OpenAI API key.
  -d, --delete                   Delete the local image after pushing
  -h, --help                     help for az
  -g, --registry-name string     Azure Container Registry name (required)
  -r, --resource-group string    Azure resource group name (required)
  -s, --subscription-id string   Azure subscription ID (required)
```

### SEE ALSO

* [smurf sdkr push](smurf_sdkr_push.md)	 - Push cmd helps to push images to Docker Hub, ACR, GCR, ECR

