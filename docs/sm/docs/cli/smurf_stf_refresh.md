## smurf stf refresh

Update the state file of your infrastructure

```
smurf stf refresh [flags]
```

### Examples

```

    # Basic refresh
    smurf stf refresh

    # Refresh with a specific directory
    smurf stf refresh --dir=path/to/terraform/code

    # Refresh with variables
    smurf stf refresh --var="region=us-west-2"

    # Refresh with variable file
    smurf stf refresh --var-file="prod.tfvars"

    # Refresh without state locking
    smurf stf refresh --lock=false
    
```

### Options

```
      --ai                     To enable AI help mode, export the OPENAI_API_KEY environment variable with your OpenAI API key.
      --dir string             Specify the Terraform directory (default ".")
  -h, --help                   help for refresh
      --lock                   Lock the state file when running operation (defaults to true) (default true)
      --var stringArray        Set a variable in 'name=value' format
      --var-file stringArray   Path to a Terraform variable file
```

### SEE ALSO

* [smurf stf](smurf_stf.md)	 - Subcommand for Terraform-related actions

