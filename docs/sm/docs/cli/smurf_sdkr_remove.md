## smurf sdkr remove

Remove a Docker image from the local system.

```
smurf sdkr remove [IMAGE_NAME[:TAG]] [flags]
```

### Examples

```

  smurf sdkr remove my-image:latest
  smurf sdkr remove
  # In the second example, it will read IMAGE_NAME from the config file

```

### Options

```
      --ai     To enable AI help mode, export the OPENAI_API_KEY environment variable with your OpenAI API key.
  -h, --help   help for remove
```

### SEE ALSO

* [smurf sdkr](smurf_sdkr.md)	 - Subcommand for Docker-related actions

