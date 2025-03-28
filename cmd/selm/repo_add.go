package selm

import (
	"github.com/clouddrove/smurf/configs"
	"github.com/clouddrove/smurf/internal/helm"
	"github.com/spf13/cobra"
)

var repoAddCmd = &cobra.Command{
	Use:   "add [NAME] [URL]",
	Short: "Add a chart repository",
	Long: `Add a chart repository to your local repository list.
The repository can be accessed by its name in other commands.`,
	Args: cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		var username, password, certFile, keyFile, caFile string
		username = configs.Username
		password = configs.Password
		certFile = configs.CertFile
		keyFile = configs.KeyFile
		caFile = configs.CaFile
		return helm.Repo_Add(args, username, password, certFile, keyFile, caFile)
	},
	Example: `
  # Add a chart repository
  smurf selm repo add prometheus https://prometheus-community.github.io/helm-charts

  # Add a private chart repository with auth
  smurf selm repo add myrepo https://charts.example.com --username myuser --password mypass`,
}

func init() {
	// Add repo command flags
	repoAddCmd.Flags().StringVar(&configs.Username, "username", "", "Chart repository username")
	repoAddCmd.Flags().StringVar(&configs.Password, "password", "", "Chart repository password")
	repoAddCmd.Flags().StringVar(&configs.CertFile, "cert-file", "", "Identify HTTPS client using this SSL certificate file")
	repoAddCmd.Flags().StringVar(&configs.KeyFile, "key-file", "", "Identify HTTPS client using this SSL key file")
	repoAddCmd.Flags().StringVar(&configs.CaFile, "ca-file", "", "Verify certificates of HTTPS-enabled servers using this CA bundle")

	// Add commands to root
	repoCmd.AddCommand(repoAddCmd)
}
