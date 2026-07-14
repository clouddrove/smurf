## smurf stf show

Show Terraform state or saved plan details

```
smurf stf show [plan-file] [flags]
```

### Examples

```

    # Show current state
    smurf stf show

    # Show state in JSON format
    smurf stf show --json

    # Show saved plan file
    smurf stf show plan.out
    smurf stf show --plan=plan.out

    # Show plan file in JSON format
    smurf stf show plan.out --json

    # Show specific resource from state
    smurf stf show --resource=aws_instance.web
    smurf stf show --resource=module.vpc

    # Show resource in JSON format
    smurf stf show --resource=aws_instance.web --json

    # Show state from custom directory
    smurf stf show --dir=environments/prod

    # Show state with variables
    smurf stf show --var="region=us-west-2"
    smurf stf show --var-file=vars.tfvars
    
```

### Options

```
      --ai                     To enable AI help mode, export the OPENAI_API_KEY environment variable with your OpenAI API key.
      --dir string             Specify the directory containing Terraform files (default ".")
  -h, --help                   help for show
      --json                   Output in JSON format
      --plan string            Path to a saved plan file to show
      --resource string        Show specific resource by address (e.g., aws_instance.web)
      --var stringArray        Specify a variable in 'NAME=VALUE' format
      --var-file stringArray   Specify a file containing variables
```

### SEE ALSO

* [smurf stf](smurf_stf.md)	 - Subcommand for Terraform-related actions

