## smurf selm uninstall

Uninstall a Helm release and all its resources

### Synopsis

This command uninstalls a Helm release and ensures all associated Kubernetes resources
are properly deleted. It automatically handles cleanup of remaining resources.

```
smurf selm uninstall [NAME] [flags]
```

### Examples

```

smurf selm uninstall my-release
# Uninstalls 'my-release' from the 'default' namespace

smurf selm uninstall my-release -n my-namespace
# Uninstalls 'my-release' from the 'my-namespace' namespace

smurf selm uninstall
# Reads NAME from the config file and uninstalls from the specified namespace or 'default' if not set

```

### Options

```
      --ai                 To enable AI help mode, export the OPENAI_API_KEY environment variable with your OpenAI API key.
      --cascade string     Delete cascading policy (background, foreground, orphan) (default "background")
  -h, --help               help for uninstall
  -n, --namespace string   Namespace of the release
      --no-hooks           Prevent hooks from running during uninstall
      --timeout duration   Time to wait for deletion (default 10m0s)
```

### SEE ALSO

* [smurf selm](smurf_selm.md)	 - Subcommand for Helm-related actions

