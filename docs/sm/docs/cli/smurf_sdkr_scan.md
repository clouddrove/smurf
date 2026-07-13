## smurf sdkr scan

Scan a Docker image for known vulnerabilities.

```
smurf sdkr scan [IMAGE_NAME[:TAG]] [flags]
```

### Examples

```

 smurf sdkr scan my-image:latest
 smurf sdkr scan
 # In the second example, it will read IMAGE_NAME from the config file

```

### Options

```
      --ai     To enable AI help mode, export the OPENAI_API_KEY environment variable with your OpenAI API key.
  -h, --help   help for scan
```

### SEE ALSO

* [smurf sdkr](smurf_sdkr.md)	 - Subcommand for Docker-related actions

