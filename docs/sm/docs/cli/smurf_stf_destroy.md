## smurf stf destroy

Destroy the Terraform Infrastructure

```
smurf stf destroy [flags]
```

### Examples

```

	# simple smurf stf destroy commad
	smurf stf destroy

	# Skip approval prompt
	smurf stf destroy --auto-approve

	# Specify a custom directory
	smurf stf destroy --dir=/path/to/terraform

	# Use variable files
	smurf stf destroy --var-file=production.tfvars
	smurf stf destroy --var-file=common.tfvars --var-file=production.tfvars

	# Use variables
	smurf stf destroy --var="environment=staging"
	
	# Combined usage
	smurf stf destroy --auto-approve --var-file=prod.tfvars --var="force_destroy=true"

```

### Options

```
      --ai                     To enable AI help mode, export the OPENAI_API_KEY environment variable with your OpenAI API key.
      --auto-approve           Skip interactive approval of plan before destroying
      --dir string             Specify the directory containing Terraform configuration (default ".")
  -h, --help                   help for destroy
      --lock                   Hold a state lock during the operation (disable with --lock=false) (default true)
      --var stringArray        Specify a variable in 'NAME=VALUE' format
      --var-file stringArray   Specify a file containing variables
```

### SEE ALSO

* [smurf stf](smurf_stf.md)	 - Subcommand for Terraform-related actions

