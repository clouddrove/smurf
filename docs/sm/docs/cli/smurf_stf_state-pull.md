## smurf stf state-pull

Pull and display the current remote state

### Synopsis

Fetch the current state from the remote backend and display it in JSON format.
This is useful for inspecting the state stored in remote backends like S3, GCS, or Terraform Cloud.

```
smurf stf state-pull [flags]
```

### Examples

```

    # Pull and display remote state
    smurf stf state-pull

    # Pull state from specific directory
    smurf stf state-pull --dir=path/to/terraform/code

    # Pull state with AI assistance on errors
    smurf stf state-pull --ai --dir=prod/environment

    # Save pulled state to a file
    smurf stf state-pull > remote-state.json

    # Pretty print with jq (if installed)
    smurf stf state-pull | jq '.'
    
```

### Options

```
      --ai           To enable AI help mode, export the OPENAI_API_KEY environment variable with your OpenAI API key.
      --dir string   Specify the Terraform directory (default ".")
  -h, --help         help for state-pull
```

### SEE ALSO

* [smurf stf](smurf_stf.md)	 - Subcommand for Terraform-related actions

