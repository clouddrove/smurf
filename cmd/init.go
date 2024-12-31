package cmd

import (
	"fmt"
	"os"
	"gopkg.in/yaml.v2"
	"github.com/spf13/cobra"
	"github.com/clouddrove/smurf/configs"
)

// generateConfigCmd represents the init command to generate a smurf.yaml configuration file with empty values
var generateConfig = &cobra.Command{
	Use:   "init",
	Short: "Generate a smurf.yaml configuration file with empty values",
	RunE: func(cmd *cobra.Command, args []string) error {
		config := map[string]interface{}{
			"sdkr": map[string]interface{}{
				"docker_password":            "",
				"docker_username":            "",
				"provisionAcrRegistryName":   "",
				"provisionAcrResourceGroup":  "",
				"provisionAcrSubscriptionID": "",
				"provisionEcrRegion":         "",
				"provisionEcrRepository":     "",
				"provisionGcrProjectID":      "",
				"google_application_credentials": "",
				"imageName":                  "",
				"targetImageTag":                  "",
			},
			"selm": map[string]interface{}{
				"releaseName":                "",
				"namespace":                  "",
				"chartName":                  "",
				"revision":                   0,
			},
		}

		file, err := os.Create(configs.FileName)
		if err != nil {
			return fmt.Errorf("error creating YAML file: %v", err)
		}
		defer file.Close()

		data, err := yaml.Marshal(&config)
		if err != nil {
			return fmt.Errorf("error marshaling data to YAML: %v", err)
		}

		if _, err := file.Write(data); err != nil {
			return fmt.Errorf("error writing to YAML file: %v", err)
		}

		fmt.Println("smurf.yaml configuration file generated successfully with empty values.")
		return nil
	},
}

func init() {
	RootCmd.AddCommand(generateConfig)
}