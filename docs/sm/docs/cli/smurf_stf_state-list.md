## smurf stf state-list

List resources in the Terraform state

```
smurf stf state-list [flags]
```

### Examples

```

    # List all resources in state
    smurf stf state-list

    # List resources in a specific directory
    smurf stf state-list --dir=path/to/terraform/code

    # List resources as a JSON array
    smurf stf state-list -o json
    
```

### Options

```
      --ai              To enable AI help mode, export the OPENAI_API_KEY environment variable with your OpenAI API key.
      --dir string      Specify the Terraform directory (default ".")
  -h, --help            help for state-list
  -o, --output string   output format (table|json) (default "table")
```

### SEE ALSO

* [smurf stf](smurf_stf.md)	 - Subcommand for Terraform-related actions

