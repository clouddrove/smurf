## smurf sdkr push aws

Push Docker images to ECR

```
smurf sdkr push aws [IMAGE_NAME] [flags]
```

### Examples

```

  # IMAGE_NAME can be in the form:
  #   123456789012.dkr.ecr.us-east-1.amazonaws.com/my-repo:python

  smurf sdkr push aws 123456789012.dkr.ecr.us-east-1.amazonaws.com/my-repo:python
  smurf sdkr push aws 123456789012.dkr.ecr.us-east-1.amazonaws.com/repo-name:python --delete

```

### Options

```
      --ai       To enable AI help mode, export the OPENAI_API_KEY environment variable with your OpenAI API key.
  -d, --delete   Delete the local image after pushing
  -h, --help     help for aws
```

### SEE ALSO

* [smurf sdkr push](smurf_sdkr_push.md)	 - Push cmd helps to push images to Docker Hub, ACR, GCR, ECR

