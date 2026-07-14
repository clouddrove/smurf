## smurf stf init

Initialize Terraform

### Synopsis

Initialize a Terraform working directory.
		   This command performs several initialization steps including:
        		- Download and install provider plugins
				- Initialize backend configuration
				- Download and install modules
				- Set up workspace configuration

```
smurf stf init [flags]
```

### Examples

```

 # Basic initialization
  smurf stf init

  # Initialize with backend configuration file
  smurf stf init --backend-config=backend.hcl

  # Initialize with multiple backend config files
  smurf stf init --backend-config=backend.hcl --backend-config=prod-backend.hcl

  # Reconfigure backend (ignore existing config)
  smurf stf init --reconfigure

  # Migrate state to new backend
  smurf stf init --migrate-state

  # Reconfigure and migrate state
  smurf stf init --reconfigure --migrate-state

  # Initialize from module source
  smurf stf init --from-module=github.com/terraform-aws-modules/terraform-aws-vpc

  # Skip downloading modules
  smurf stf init --get=false

```

### Options

```
      --ai                           Enable AI help mode (requires OPENAI_API_KEY)
      --backend                      Configure backend (disable with --backend=false) (default true)
      --backend-config stringArray   Path to backend configuration file (can be used multiple times)
      --dir string                   Directory containing Terraform files (default is current directory) (default ".")
      --force-copy                   Suppress prompts about copying state data during backend migration
      --from-module string           Copy the source module into the target directory
      --get                          Download and install modules (default true)
  -h, --help                         help for init
      --migrate-state                Migrate existing state to new backend
      --reconfigure                  Reconfigure backend, ignoring existing configuration
      --upgrade                      Upgrade installed modules and plugins
```

### SEE ALSO

* [smurf stf](smurf_stf.md)	 - Subcommand for Terraform-related actions

