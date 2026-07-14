## smurf stf provision

Its the combination of init, plan, apply, output for Terraform

```
smurf stf provision [flags]
```

### Examples

```

	# Prompts for interactive approval before applying
	smurf stf provision

	# Skip interactive approval of plan before applying
	smurf stf provision --dir=/path/to/terraform/files --auto-approve
	
```

### Options

```
      --ai                     To enable AI help mode, export the OPENAI_API_KEY environment variable with your OpenAI API key.
      --auto-approve           Skip interactive approval of plan before applying
      --dir string             Specify the directory for Terraform operations
  -h, --help                   help for provision
      --lock                   Hold a state lock during the operation (disable with --lock=false) (default true)
      --out string             Path to save the generated execution plan
      --upgrade                Upgrade the Terraform modules and plugins to the latest versions
      --var stringArray        Specify a variable in 'NAME=VALUE' format
      --var-file stringArray   Specify a file containing variables
```

### SEE ALSO

* [smurf stf](smurf_stf.md)	 - Subcommand for Terraform-related actions

