## smurf stf fmt

Format the Terraform Infrastructure

```
smurf stf fmt [flags]
```

### Examples

```

	smurf stf fmt
	smurf stf fmt --timeout 30s
	smurf stf fmt --recursive --timeout 2m
	
```

### Options

```
  -h, --help               help for fmt
  -r, --recursive          Run the command recursively on all subdirectories. By default, only the given directory (or current directory) is processed.
  -t, --timeout duration   Timeout for the formatting process (e.g., 30s, 2m, 1h). Zero means no timeout.
```

### SEE ALSO

* [smurf stf](smurf_stf.md)	 - Subcommand for Terraform-related actions

