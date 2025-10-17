package sdkr

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

// defaultYamlContent contains the initial smurf.yaml structure
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
`

// sdkrCreateCmd defines the "smurf sdkr create" command
var sdkrCreateCmd = &cobra.Command{
	Use:   "init",
	Short: "Create a default smurf.yaml file with sdkr configuration",
	Long: `This command generates a smurf.yaml file in the current working directory
with default sdkr configuration placeholders.`,
	SilenceUsage: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		fileName := "smurf.yaml"

		// Check if file already exists to prevent accidental overwrite
		if _, err := os.Stat(fileName); err == nil {
			return fmt.Errorf("%s already exists. Delete or rename it before creating a new one", fileName)
		}

		// Write default YAML content
		err := os.WriteFile(fileName, []byte(defaultYamlContent), 0644)
		if err != nil {
			return fmt.Errorf("failed to create %s: %v", fileName, err)
		}

		fmt.Printf("✅ %s created successfully.\n", fileName)
		return nil
	},
}

func init() {
	sdkrCmd.AddCommand(sdkrCreateCmd)
}
