package selm

import (
	"os"
	"path/filepath"

	"github.com/clouddrove/smurf/configs"
	"github.com/clouddrove/smurf/internal/helm"
	"github.com/spf13/cobra"
)

var pullCmd = &cobra.Command{
	Use:   "pull [CHART]",
	Short: "Download a chart from a repository",
	Long: `Download a chart from a repository and save it to the local directory.
	
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
  smurf selm pull repo/chart-name --prov --keyring ~/.gnupg/pubring.gpg`,
	Args:         cobra.ExactArgs(1),
	SilenceUsage: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		chartRef := args[0]
		return helm.Pull(chartRef, configs.Version, configs.Destination, configs.Untar,
			configs.UntarDir, configs.Verify, configs.Keyring, configs.RepoURL,
			configs.Username, configs.Password, configs.CertFile, configs.KeyFile,
			configs.CaFile, configs.Insecure, configs.PlainHttp, configs.PassCredentials,
			configs.Devel, configs.Prov, configs.HelmConfigDir)
	},
}

func init() {
	// Add all pull command flags with proper names
	pullCmd.Flags().StringVarP(&configs.Version, "version", "v", "", "Specify the version constraint for the chart")
	pullCmd.Flags().StringVarP(&configs.Destination, "destination", "d", ".", "Location to write the chart")
	pullCmd.Flags().BoolVar(&configs.Untar, "untar", false, "If set to true, will untar the chart after downloading it")
	pullCmd.Flags().StringVar(&configs.UntarDir, "untardir", ".", "If untar is specified, this flag specifies the name of the directory into which the chart is expanded")
	pullCmd.Flags().BoolVar(&configs.Verify, "verify", false, "Verify the package against its signature")
	pullCmd.Flags().StringVar(&configs.Keyring, "keyring", defaultKeyring(), "Location of public keys used for verification")
	pullCmd.Flags().StringVar(&configs.RepoURL, "repo", "", "Chart repository URL where to locate the requested chart")
	pullCmd.Flags().StringVar(&configs.Username, "username", "", "Chart repository username")
	pullCmd.Flags().StringVar(&configs.Password, "password", "", "Chart repository password")
	pullCmd.Flags().StringVar(&configs.CertFile, "cert-file", "", "Identify HTTPS client using this SSL certificate file")
	pullCmd.Flags().StringVar(&configs.KeyFile, "key-file", "", "Identify HTTPS client using this SSL key file")
	pullCmd.Flags().StringVar(&configs.CaFile, "ca-file", "", "Verify certificates of HTTPS-enabled servers using this CA bundle")
	pullCmd.Flags().BoolVar(&configs.Insecure, "insecure-skip-tls-verify", false, "Skip tls certificate checks for the chart download")
	pullCmd.Flags().BoolVar(&configs.PlainHttp, "plain-http", false, "Use HTTP instead of HTTPS for chart download")
	pullCmd.Flags().BoolVar(&configs.PassCredentials, "pass-credentials", false, "Pass credentials to all domains")
	pullCmd.Flags().BoolVar(&configs.Devel, "devel", false, "Use development versions (alpha, beta, and release candidate releases)")
	pullCmd.Flags().BoolVar(&configs.Prov, "prov", false, "Fetch the provenance file, but don't perform verification")
	pullCmd.Flags().StringVar(&configs.HelmConfigDir, "helm-config", "", "Helm configuration directory")

	// Add to selm command
	selmCmd.AddCommand(pullCmd)
}

func defaultKeyring() string {
	return filepath.Join(os.Getenv("HOME"), ".gnupg", "pubring.gpg")
}
