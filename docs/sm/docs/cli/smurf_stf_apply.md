## smurf stf apply

Apply the changes required to reach the desired state of Terraform Infrastructure

```
smurf stf apply [plan-file] [flags]
```

### Examples

```

	# Apply command
	smurf stf apply

	# Specify variables
	smurf stf apply --var="region=us-west-2"

	# Skip approval prompt
	smurf stf apply --auto-approve

	# Apply using a plan file (automatically skips confirmation)
	smurf stf apply plan.out

	# Apply using a plan file with variables
	smurf stf apply plan.out --var="region=us-west-2"

	# Specify multiple variables
	smurf stf apply --var="region=us-west-2" --var="instance_type=t2.micro"

	# Specify a custom directory
	smurf stf apply --dir=/path/to/terraform/files

	# Target specific resources
	smurf stf apply --target=aws_instance.web
	smurf stf apply --target=module.vpc
	smurf stf apply --target=aws_instance.web --target=aws_security_group.web

	# Use custom state file
	smurf stf apply --state=/path/to/terraform.tfstate
	smurf stf apply --state=prod.tfstate
	
```

### Options

```
      --ai                     To enable AI help mode, export the OPENAI_API_KEY environment variable with your OpenAI API key.
      --auto-approve           Skip interactive approval of plan before applying
      --dir string             Specify the directory containing Terraform files (default ".")
  -h, --help                   help for apply
      --lock                   Hold a state lock during the operation (disable with --lock=false) (default true)
      --plan string            Path to a plan file to apply (skips approval prompt)
      --state string           Path to read and save the Terraform state
      --target stringArray     Target specific resources, modules, or resources in modules
      --var stringArray        Specify a variable in 'NAME=VALUE' format
      --var-file stringArray   Specify a file containing variables
```

### SEE ALSO

* [smurf stf](smurf_stf.md)	 - Subcommand for Terraform-related actions

