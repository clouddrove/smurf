## smurf stf state-rm

Remove resources from the Terraform state

### Synopsis

Remove one or more resources from the Terraform state file. This command is useful for unmanaging resources without destroying them.

```
smurf stf state-rm [address...] [flags]
```

### Examples

```

    # Remove a single resource from state
    smurf stf state-rm aws_instance.example

    # Remove multiple resources
    smurf stf state-rm aws_instance.example aws_vpc.main

    # Remove resource from specific module
    smurf stf state-rm module.vpc.aws_subnet.private

    # Remove all resources of a type (using wildcard)
    smurf stf state-rm 'aws_instance.*'

    # Remove resource in a specific directory
    smurf stf state-rm --dir=path/to/terraform/code aws_instance.example

    # Remove without creating a backup
    smurf stf state-rm --backup=false aws_instance.example
    
```

### Options

```
      --ai           To enable AI help mode, export the OPENAI_API_KEY environment variable with your OpenAI API key.
      --backup       Create a backup of the state file before removal (default true)
      --dir string   Specify the Terraform directory (default ".")
  -h, --help         help for state-rm
```

### SEE ALSO

* [smurf stf](smurf_stf.md)	 - Subcommand for Terraform-related actions

