## smurf stf state-push

Push local state to remote backend

### Synopsis

Push the local state file to the remote backend storage.
This command should be used with caution as it can overwrite remote state.

```
smurf stf state-push [flags]
```

### Examples

```

    # Push local state to remote backend (will show diff first)
    smurf stf state-push

    # Force push without confirmation
    smurf stf state-push --force

    # Push without creating backup
    smurf stf state-push --backup=false

    # Push from specific directory
    smurf stf state-push --dir=path/to/terraform/code

    # Push with lock timeout
    smurf stf state-push --lock-timeout=60s

    # Push with AI assistance on errors
    smurf stf state-push --ai --force
    
```

### Options

```
      --ai                    To enable AI help mode, export the OPENAI_API_KEY environment variable with your OpenAI API key.
      --backup                Create backup of remote state before pushing (default true)
      --dir string            Specify the Terraform directory (default ".")
      --force                 Force push without confirmation
  -h, --help                  help for state-push
      --lock                  Lock the state file when pushing (default true)
      --lock-timeout string   Duration to retry acquiring a state lock (default "0s")
```

### SEE ALSO

* [smurf stf](smurf_stf.md)	 - Subcommand for Terraform-related actions

