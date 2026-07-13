## smurf selm create

Create a new Helm chart in the specified directory.

```
smurf selm create [NAME] [flags]
```

### Examples

```

smurf selm create mychart
# In this example, it will create 'mychart' in the current directory
smurf selm create
# In this example, it will create a chart with the name specified in the config in the current directory

```

### Options

```
  -d, --directory string     Specify the directory to create the Helm chart in (default ".")
  -h, --help                 help for create
  -f, --values stringArray   Specify values in a YAML file
```

### SEE ALSO

* [smurf selm](smurf_selm.md)	 - Subcommand for Helm-related actions

