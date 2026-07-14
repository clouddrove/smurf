## smurf init

Generate a smurf.yaml configuration file with sdkr and selm sections

### Synopsis

Generate a smurf.yaml configuration file in the current working directory,
pre-filled with placeholder values for both the sdkr and selm sections.

Refuses to run if smurf.yaml already exists, so it never overwrites an
existing configuration. Use "smurf sdkr init" or "smurf selm init" instead
if you only want to scaffold one section.

```
smurf init [flags]
```

### Options

```
  -h, --help   help for init
```

### SEE ALSO

* [smurf](smurf.md)	 - Smurf is a tool for automating common commands across Terraform, Docker, and more

