## smurf selm provision

Combination of install, upgrade, lint, and template for Helm

```
smurf selm provision [RELEASE] [CHART] [flags]
```

### Examples

```

smurf selm provision my-release ./mychart
smurf selm provision
# In this example, it will read RELEASE and CHART from the config file
smurf selm provision my-release ./mychart -n custom-namespace

```

### Options

```
      --ai                 To enable AI help mode, export the OPENAI_API_KEY environment variable with your OpenAI API key.
  -h, --help               help for provision
  -n, --namespace string   Specify the namespace to provision the Helm chart
```

### SEE ALSO

* [smurf selm](smurf_selm.md)	 - Subcommand for Helm-related actions

