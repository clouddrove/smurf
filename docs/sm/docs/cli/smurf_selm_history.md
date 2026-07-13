## smurf selm history

Show revision history for a release

```
smurf selm history [RELEASE] [flags]
```

### Examples

```

# Show history for a release
smurf selm history my-release

# Show last 5 revisions
smurf selm history my-release --max 5

```

### Options

```
      --ai                 To enable AI help mode, export the OPENAI_API_KEY environment variable with your OpenAI API key.
  -h, --help               help for history
      --max int            maximum number of revisions to show (default 256)
  -n, --namespace string   namespace of the release
```

### SEE ALSO

* [smurf selm](smurf_selm.md)	 - Subcommand for Helm-related actions

