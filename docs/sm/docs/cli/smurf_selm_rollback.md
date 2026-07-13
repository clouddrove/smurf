## smurf selm rollback

Roll back a release to a previous revision

### Synopsis

Roll back a release to a previous revision.
The first argument is the name of the release to roll back, and the second is the revision number to roll back to.

```
smurf selm rollback [RELEASE] [REVISION] [flags]
```

### Examples

```

      smurf selm rollback nginx 2
      smurf selm rollback nginx 2 --namespace mynamespace --debug
      smurf selm rollback nginx 2 --force --timeout 600
      smurf selm rollback
	  smurf selm rollback --history-max 5
      # In this example, it will read RELEASE and REVISION from the config file
    
```

### Options

```
      --ai                 To enable AI help mode, export the OPENAI_API_KEY environment variable with your OpenAI API key.
      --debug              Enable debug logging
      --force              Force rollback even if there are conflicts
  -h, --help               help for rollback
      --history-max int    Limit the maximum number of revisions saved per release (default 10)
  -n, --namespace string   Namespace of the release (default "default")
      --timeout int        Timeout for the rollback operation in seconds (default 300)
      --wait               Wait until all resources are rolled back successfully (default true)
```

### SEE ALSO

* [smurf selm](smurf_selm.md)	 - Subcommand for Helm-related actions

