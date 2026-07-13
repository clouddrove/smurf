## smurf

Smurf is a tool for automating common commands across Terraform, Docker, and more

### Synopsis

Smurf is a command-line interface built with Cobra, designed to simplify and automate commands for essential tools like Terraform and Docker. It provides intuitive, unified commands to execute Terraform plans, Docker container management, and other DevOps tasks seamlessly from one interface.
			If you are facing issues, unable to find a command, or need help, please create an issue at: https://github.com/clouddrove/smurf/issues

```
smurf [flags]
```

### Examples

```
smurf --help
```

### Options

```
  -h, --help   help for smurf
```

### SEE ALSO

* [smurf completion](smurf_completion.md)	 - Generate the autocompletion script for the specified shell
* [smurf deploy](smurf_deploy.md)	 - Deploy builds and pushes Docker image as per smurf.yaml, then optionally runs Helm deploy.
* [smurf init](smurf_init.md)	 - Generate a smurf.yaml configuration file with sdkr and selm sections
* [smurf sdkr](smurf_sdkr.md)	 - Subcommand for Docker-related actions
* [smurf selm](smurf_selm.md)	 - Subcommand for Helm-related actions
* [smurf stf](smurf_stf.md)	 - Subcommand for Terraform-related actions
* [smurf version](smurf_version.md)	 - Print detailed version information

