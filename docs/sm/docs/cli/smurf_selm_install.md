## smurf selm install

Install a Helm chart into a Kubernetes cluster.

```
smurf selm install [RELEASE] [CHART] [flags]
```

### Examples

```

  smurf selm install my-release ./mychart
  smurf selm install my-release ./mychart -n my-namespace
  smurf selm install my-release ./mychart -f values.yaml
  smurf selm install my-release ./mychart --timeout=600
  smurf selm install prometheus-11 prometheus --repo https://prometheus-community.github.io/helm-charts --version 13.0.0
  smurf selm install prometheus prometheus-community/prometheus
  smurf selm install my-release ./mychart --set key1=val1 --set key2=val2
  smurf selm install my-release ./mychart --set-literal myPassword='MySecurePass!'
  smurf selm install --wait  # Wait for resources to be ready
  smurf selm install
  # In the last example, it will read RELEASE and CHART from the config file
  
```

### Options

```
      --ai                    To enable AI help mode, export the OPENAI_API_KEY environment variable with your OpenAI API key.
      --atomic                If set, installation process purges chart on fail
      --debug                 Enable verbose output
  -h, --help                  help for install
  -n, --namespace string      Specify the namespace to install the Helm chart
      --repo string           Specify the chart repository URL for remote charts
      --set strings           Set values on the command line
      --set-literal strings   Set literal values on the command line
      --timeout int           Specify the timeout in seconds to wait for any individual Kubernetes operation (default 600)
  -f, --values stringArray    Specify values in a YAML file
      --version string        Specify the chart version to install
      --wait                  Wait for all resources to be ready before marking the release as successful (default true)
```

### SEE ALSO

* [smurf selm](smurf_selm.md)	 - Subcommand for Helm-related actions

