package selm

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

// defaultYamlContent contains the initial smurf.yaml structure
var defaultYamlContent = `selm:
  releaseName: "Release Name"
  namespace: "Name Space"
  chartName: "Chart Name"
  revision: 0
`

// sdkrCreateCmd defines the "smurf sdkr create" command
var selmCreateCmd = &cobra.Command{
	Use:   "init",
	Short: "Create a default smurf.yaml file with selm configuration",
	Long: `This command generates a smurf.yaml file in the current working directory
with default selm configuration placeholders.`,
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
	selmCmd.AddCommand(selmCreateCmd)
}
