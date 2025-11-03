package sdkr

import (
	"github.com/clouddrove/smurf/internal/utils"
	"github.com/spf13/cobra"
)

// defaultYamlContent contains the initial smurf.yaml structure for sdkr
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
`

// sdkrCreateCmd defines the "smurf sdkr init" command
var sdkrCreateCmd = &cobra.Command{
	Use:   "init",
	Short: "Create a default smurf.yaml file with sdkr configuration",
	Long: `This command generates a smurf.yaml file in the current working directory
with default sdkr configuration placeholders.`,
	SilenceUsage: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		return utils.CreateYamlFile("smurf.yaml", defaultYamlContent)
	},
}

func init() {
	sdkrCmd.AddCommand(sdkrCreateCmd)
}
