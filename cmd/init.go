package cmd

import (
	"github.com/clouddrove/smurf/internal/utils"
	"github.com/spf13/cobra"
)

// defaultYamlContent is the full smurf.yaml scaffold: the union of the sdkr
// section written by "smurf sdkr init" and the selm section written by
// "smurf selm init", so the three init commands stop producing conflicting
// schemas.
var defaultYamlContent = `sdkr:
  docker_username: "my-docker-username"
  docker_password: "my-docker-password"
  github_username: "my-github_username"
  github_token: "my-github_token"
  provisionAcrRegistryName: "myacrregistry"
  provisionAcrResourceGroup: "my-resource-group"
  provisionAcrSubscriptionID: "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  provisionGcrProjectID: "my-gcr-project-id"
  google_application_credentials: "/path/to/service-account-key.json"
  imageName: "my-application"
  targetImageTag: "v1.0.0"
  awsAccessKey: ""
  awsSecretKey: ""
  awsRegion: "us-east-1"
  dockerfile: ""
  awsECR: false
  dockerHub: false
  ghcrRepo: false
  gcpRepo: false
selm:
  deployHelm: false
  releaseName: "Release Name"
  namespace: "Name Space"
  chartName: "Chart Name"
  fileName: ""
  revision: 0
`

// generateConfig represents the "smurf init" command, which generates a
// smurf.yaml configuration file containing both the sdkr and selm sections.
var generateConfig = &cobra.Command{
	Use:   "init",
	Short: "Generate a smurf.yaml configuration file with sdkr and selm sections",
	Long: `Generate a smurf.yaml configuration file in the current working directory,
pre-filled with placeholder values for both the sdkr and selm sections.

Refuses to run if smurf.yaml already exists, so it never overwrites an
existing configuration. Use "smurf sdkr init" or "smurf selm init" instead
if you only want to scaffold one section.`,
	SilenceUsage: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		return utils.CreateYamlFile("smurf.yaml", defaultYamlContent)
	},
}

func init() {
	RootCmd.AddCommand(generateConfig)
}
