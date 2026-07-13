## smurf selm status

Status of a Helm release.

```
smurf selm status [NAME] [flags]
```

### Examples

```

	smurf selm status my-release
	# In this example, it will fetch the status of 'my-release' in the 'default' namespace

	smurf selm status my-release -n my-namespace
	# In this example, it will fetch the status of 'my-release' in the 'my-namespace' namespace

	smurf selm status
	# In this example, it will read the release name from the config file and fetch its status
	
```

### Options

```
      --ai                 To enable AI help mode, export the OPENAI_API_KEY environment variable with your OpenAI API key.
  -h, --help               help for status
  -n, --namespace string   Specify the namespace to get status of the Helm chart
```

### SEE ALSO

* [smurf selm](smurf_selm.md)	 - Subcommand for Helm-related actions

