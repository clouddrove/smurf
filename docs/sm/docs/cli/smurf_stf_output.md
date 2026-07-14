## smurf stf output

Generate output for the current state of Terraform Infrastructure

```
smurf stf output [flags]
```

### Examples

```

	smurf stf output
	smurf stf output --dir <terraform-directory>
	smurf stf output -o json
	
```

### Options

```
      --ai              To enable AI help mode, export the OPENAI_API_KEY environment variable with your OpenAI API key.
      --dir string      Specify the Terraform directory (default ".")
  -h, --help            help for output
  -o, --output string   output format (table|json) (default "table")
```

### SEE ALSO

* [smurf stf](smurf_stf.md)	 - Subcommand for Terraform-related actions

