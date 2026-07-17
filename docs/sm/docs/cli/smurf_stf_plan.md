## smurf stf plan

Generate and show an execution plan for Terraform

```
smurf stf plan [flags]
```

### Examples

```

    smurf stf plan

    # Specify variables
    smurf stf plan --var="region=us-west-2"

    # Specify multiple variables
    smurf stf plan --var="region=us-west-2" --var="instance_type=t2.micro"

    # Specify a custom directory
    smurf stf plan --dir=/path/to/terraform/files

    # Plan for destroy
    smurf stf plan --destroy

    # Target specific resources
    smurf stf plan --target=aws_instance.web
    smurf stf plan --target=module.vpc
    smurf stf plan --target=aws_instance.web --target=aws_security_group.web

    # Skip refresh
    smurf stf plan --refresh=false

    # Use custom state file
    smurf stf plan --state=/path/to/terraform.tfstate
    smurf stf plan --state=prod.tfstate

    # Combine with other flags
    smurf stf plan --target=aws_instance.web --destroy --var="instance_type=t2.micro" --refresh=false --state=prod.tfstate
    smurf stf plan --out=prod.plan --var-file=vars.tfvars

    # CI/CD detailed exit codes (0 = no changes, 1 = error, 2 = changes pending)
    smurf stf plan --detailed-exitcode --out=tfplan --var-file=vars.tfvars
    
```

### Options

```
      --ai                     To enable AI help mode, export the OPENAI_API_KEY environment variable with your OpenAI API key.
      --destroy                Generate a destroy plan
      --detailed-exitcode      Return exit code 2 when changes are pending (0 = no changes, 1 = error)
      --dir string             Specify the directory containing Terraform files (default ".")
  -h, --help                   help for plan
      --out string             Path to save the generated execution plan
      --refresh                Update state prior to checking for differences (default true)
      --state string           Path to read and save the Terraform state
      --target stringArray     Target specific resources, modules, or resources in modules
      --var stringArray        Specify a variable in 'NAME=VALUE' format
      --var-file stringArray   Specify a file containing variables
```

### SEE ALSO

* [smurf stf](smurf_stf.md)	 - Subcommand for Terraform-related actions

