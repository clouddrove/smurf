## smurf selm pull

Download a chart from a repository

### Synopsis

Download a chart from a repository and save it to the local directory.
	
Examples:
  # Pull latest version
  smurf selm pull repo/chart-name
  
  # Pull specific version
  smurf selm pull repo/chart-name --version 1.2.3
  
  # Pull to specific directory
  smurf selm pull repo/chart-name --destination ./charts
  
  # Pull and untar
  smurf selm pull repo/chart-name --untar --untardir ./my-charts
  
  # Pull with authentication
  smurf selm pull repo/chart-name --username user --password pass
  
  # Pull from specific URL (bypass repo config)
  smurf selm pull https://example.com/charts/mychart-1.2.3.tgz
  
  # Pull with provenance verification
  smurf selm pull repo/chart-name --prov --keyring ~/.gnupg/pubring.gpg

```
smurf selm pull [CHART] [flags]
```

### Options

```
      --ai                         To enable AI help mode, export the OPENAI_API_KEY environment variable with your OpenAI API key.
      --ca-file string             Verify certificates of HTTPS-enabled servers using this CA bundle
      --cert-file string           Identify HTTPS client using this SSL certificate file
  -d, --destination string         Location to write the chart (default ".")
      --devel                      Use development versions (alpha, beta, and release candidate releases)
      --helm-config string         Helm configuration directory
  -h, --help                       help for pull
      --insecure-skip-tls-verify   Skip tls certificate checks for the chart download
      --key-file string            Identify HTTPS client using this SSL key file
      --keyring string             Location of public keys used for verification (default "~/.gnupg/pubring.gpg")
      --pass-credentials           Pass credentials to all domains
      --password string            Chart repository password
      --plain-http                 Use HTTP instead of HTTPS for chart download
      --prov                       Fetch the provenance file, but don't perform verification
      --repo string                Chart repository URL where to locate the requested chart
      --untar                      If set to true, will untar the chart after downloading it
      --untardir string            If untar is specified, this flag specifies the name of the directory into which the chart is expanded (default ".")
      --username string            Chart repository username
      --verify                     Verify the package against its signature
  -v, --version string             Specify the version constraint for the chart
```

### SEE ALSO

* [smurf selm](smurf_selm.md)	 - Subcommand for Helm-related actions

