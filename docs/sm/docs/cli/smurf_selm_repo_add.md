## smurf selm repo add

Add a chart repository

### Synopsis

Add a chart repository to your local repository list.
The repository can be accessed by its name in other commands.

```
smurf selm repo add [NAME] [URL] [flags]
```

### Examples

```

  # Add a chart repository
  smurf selm repo add prometheus https://prometheus-community.github.io/helm-charts

  # Add a private chart repository with auth
  smurf selm repo add myrepo https://charts.example.com --username myuser --password mypass
  
  # Add a repository with custom Helm config directory
  smurf selm repo add myrepo https://charts.example.com --helm-config /custom/path
```

### Options

```
      --ai                   To enable AI help mode, export the OPENAI_API_KEY environment variable with your OpenAI API key.
      --ca-file string       Verify certificates of HTTPS-enabled servers using this CA bundle
      --cert-file string     Identify HTTPS client using this SSL certificate file
      --helm-config string   Helm configuration directory (default: $HELM_HOME or ~/.config/helm)
  -h, --help                 help for add
      --key-file string      Identify HTTPS client using this SSL key file
      --password string      Chart repository password
      --username string      Chart repository username
```

### SEE ALSO

* [smurf selm repo](smurf_selm_repo.md)	 - Add, update, or manage chart repositories

