## smurf selm lint

Lint a Helm chart.

```
smurf selm lint [CHART] [flags]
```

### Examples

```

smurf selm lint ./mychart
smurf selm lint ./mychart -f ./my-chart/values.yaml
smurf selm lint
# In the last example, it will read CHART from the config file

```

### Options

```
      --ai                   To enable AI help mode, export the OPENAI_API_KEY environment variable with your OpenAI API key.
  -h, --help                 help for lint
  -f, --values stringArray   Specify values in a YAML file
```

### SEE ALSO

* [smurf selm](smurf_selm.md)	 - Subcommand for Helm-related actions

