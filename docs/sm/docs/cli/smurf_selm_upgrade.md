## smurf selm upgrade

Upgrade a deployed Helm chart.

```
smurf selm upgrade [NAME] [CHART] [flags]
```

### Examples

```

			# Upgrade without waiting (default behavior)
			smurf selm upgrade my-release ./mychart

			# Upgrade and wait for resources to be ready
			smurf selm upgrade my-release ./mychart --wait

			# Upgrade with custom timeout and waiting
			smurf selm upgrade my-release ./mychart --timeout 600 --wait

			# Install if not present without waiting (default)
			smurf selm upgrade my-release ./mychart --install

			# Install if not present with waiting
			smurf selm upgrade my-release ./mychart --install --wait

			# Upgrade with limited history
			smurf selm upgrade my-release ./mychart --history-max 5

			# Upgrade with all options
			smurf selm upgrade my-release ./mychart --wait --timeout 300 --history-max 3 --atomic

			# Force upgrade (Helm native behavior - forces delete/recreate)
			smurf selm upgrade my-release ./mychart --force
	
```

### Options

```
      --ai                    To enable AI help mode, export the OPENAI_API_KEY environment variable with your OpenAI API key.
      --atomic                If set, the installation process purges the chart on fail, the upgrade process rolls back changes, and the upgrade process waits for the resources to be ready
      --create-namespace      Create the namespace if it does not exist
      --debug                 Enable verbose output
      --force                 Force resource updates through delete/recreate if needed
  -h, --help                  help for upgrade
      --history-max int       Limit the maximum number of revisions saved per release (default 10)
      --install               Install the chart if it is not already installed
  -n, --namespace string      Specify the namespace to install the release into (default "default")
      --repo-url string       Helm repository URL
      --set strings           Set values on the command line (can specify multiple or separate values with commas: key1=val1,key2=val2)
      --set-literal strings   Set literal values on the command line (values are always treated as strings)
      --timeout int           Time to wait for any individual Kubernetes operation (like Jobs for hooks) (default 120)
  -f, --values strings        Specify values in a YAML file (can specify multiple)
      --version string        Helm chart version
      --wait                  Wait until all Pods, PVCs, Services, and minimum number of Pods of a Deployment are ready before marking success
```

### SEE ALSO

* [smurf selm](smurf_selm.md)	 - Subcommand for Helm-related actions

