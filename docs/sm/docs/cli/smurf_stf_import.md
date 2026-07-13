## smurf stf import

Import existing infrastructure into Terraform state

```
smurf stf import [flags] ADDRESS ID
```

### Examples

```

    # Import a basic resource
    smurf stf import aws_instance.web i-1234567890abcdef0

    # Import a resource in a module
    smurf stf import module.vpc.aws_vpc.main vpc-12345678

    # Import with custom directory
    smurf stf import --dir=/path/to/terraform/files aws_instance.web i-1234567890abcdef0

    # Import with variables
    smurf stf import --var="region=us-west-2" aws_instance.web i-1234567890abcdef0

    # Import with variable file
    smurf stf import --var-file=vars.tfvars aws_instance.web i-1234567890abcdef0

    # Import with custom state file
    smurf stf import --state=prod.tfstate aws_instance.web i-1234567890abcdef0

    # Import with config file
    smurf stf import --config=alternate.tf aws_instance.web i-1234567890abcdef0

    # Import multiple resources
    smurf stf import aws_instance.web i-1234567890abcdef0
    smurf stf import aws_security_group.web sg-12345678

    # Import without refresh (handled automatically by Terraform)
    smurf stf import --refresh=false aws_instance.web i-1234567890abcdef0

    # Allow missing resource during import
    smurf stf import --allow-missing aws_instance.web i-1234567890abcdef0

    # Complex example with multiple flags
    smurf stf import --dir=environments/prod --state=prod.tfstate --var-file=prod.tfvars \
      --allow-missing aws_instance.web i-1234567890abcdef0
    
```

### Options

```
      --ai                     To enable AI help mode, export the OPENAI_API_KEY environment variable with your OpenAI API key.
      --allow-missing          Allow import even if the configuration block is missing
      --config string          Path to a Terraform configuration file to use for import
      --dir string             Specify the directory containing Terraform files (default ".")
  -h, --help                   help for import
      --refresh                Update state prior to import (handled automatically by Terraform) (default true)
      --state string           Path to read and save the Terraform state
      --var stringArray        Specify a variable in 'NAME=VALUE' format
      --var-file stringArray   Specify a file containing variables
```

### SEE ALSO

* [smurf stf](smurf_stf.md)	 - Subcommand for Terraform-related actions

