## smurf selm repo update

Update chart repositories

### Synopsis

Update all chart repositories or a specific one if provided.

```
smurf selm repo update [REPO]... [flags]
```

### Examples

```
   # Update all chart repositories
   smurf selm repo update
   
   # Update specific repositories
   smurf selm repo update prometheus stable
```

### Options

```
      --ai                   To enable AI help mode, export the OPENAI_API_KEY environment variable with your OpenAI API key.
      --helm-config string   Helm configuration directory (default: $HELM_HOME or ~/.config/helm)
  -h, --help                 help for update
```

### SEE ALSO

* [smurf selm repo](smurf_selm_repo.md)	 - Add, update, or manage chart repositories

