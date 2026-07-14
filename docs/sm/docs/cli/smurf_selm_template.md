## smurf selm template

Render chart templates

```
smurf selm template [RELEASE] [CHART] [flags]
```

### Examples

```

smurf selm template my-release ./mychart
# In this example, it will render templates for 'my-release' in './mychart' within the 'default' namespace

smurf selm template my-release mychart --repo https://charts.clouddrove.com/
# This will pull the chart from the specified Helm repository and render it.

smurf selm template my-release ./mychart -n my-namespace -f values.yaml
# In this example, it will render templates for 'my-release' in './mychart' within 'my-namespace' using specified values files

```

### Options

```
      --ai                   To enable AI help mode, export the OPENAI_API_KEY environment variable with your OpenAI API key.
  -h, --help                 help for template
  -n, --namespace string     Specify the namespace to template the Helm chart
  -r, --repo string          Specify Helm chart repository URL
  -f, --values stringArray   Specify values in a YAML file
```

### SEE ALSO

* [smurf selm](smurf_selm.md)	 - Subcommand for Helm-related actions

