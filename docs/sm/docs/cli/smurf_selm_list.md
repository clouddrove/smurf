## smurf selm list

List Helm releases

### Synopsis

List Helm releases across namespaces with various output formats.
Defaults to showing releases in the default namespace unless specified.

```
smurf selm list [flags]
```

### Examples

```

  # List in current namespace (default)
  smurf selm list

  # List in specific namespace
  smurf selm list -n mynamespace

  # List across all namespaces
  smurf selm list -A

  # List with JSON output 
  smurf selm list -n kube-system -o json

  # List with YAML output (short flags)
  smurf selm ls -A -o yaml

```

### Options

```
      --ai                 To enable AI help mode, export the OPENAI_API_KEY environment variable with your OpenAI API key.
  -A, --all-namespaces     list across all namespaces
  -h, --help               help for list
  -n, --namespace string   namespace scope for listing (default "default")
  -o, --output string      output format (table|json|yaml) (default "table")
```

### SEE ALSO

* [smurf selm](smurf_selm.md)	 - Subcommand for Helm-related actions

