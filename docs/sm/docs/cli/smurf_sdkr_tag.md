## smurf sdkr tag

Tag a Docker image for a remote repository

```
smurf sdkr tag [SOURCE_IMAGE[:TAG]] [TARGET_IMAGE[:TAG]] [flags]
```

### Examples

```

  smurf sdkr tag my-app:latest my-org/my-app:prod
  smurf sdkr tag
  # In the second example, it reads SOURCE and TARGET from the config file

```

### Options

```
      --ai     To enable AI help mode, export the OPENAI_API_KEY environment variable with your OpenAI API key.
  -h, --help   help for tag
```

### SEE ALSO

* [smurf sdkr](smurf_sdkr.md)	 - Subcommand for Docker-related actions

