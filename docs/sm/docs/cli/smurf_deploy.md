## smurf deploy

Deploy builds and pushes Docker image as per smurf.yaml, then optionally runs Helm deploy.

### Synopsis

Deploy reads smurf.yaml and runs the full pipeline: build the Docker image,
push it to whichever registry is enabled (awsECR, dockerHub, ghcrRepo, or gcpRepo),
and then, if selm.deployHelm is true, install or upgrade the Helm release with the
new image repository and tag.

Use --timeout to control how long the push and Helm operations are allowed to run.

```
smurf deploy [flags]
```

### Examples

```

  # Run the full build, push, and Helm deploy pipeline using smurf.yaml
  smurf deploy

  # Override the timeout for push and Helm operations (in seconds)
  smurf deploy --timeout 900

```

### Options

```
  -h, --help          help for deploy
      --timeout int   Timeout in seconds for push and Helm operations (default 600)
```

### SEE ALSO

* [smurf](smurf.md)	 - Smurf is a tool for automating common commands across Terraform, Docker, and more

