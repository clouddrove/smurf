package cmd

import (
	"fmt"
	"os"
	"gopkg.in/yaml.v2"
	"github.com/spf13/cobra"
)

var generateConfig = &cobra.Command{
	Use:   "init",
	Short: "Generate a smurf.yaml configuration file with empty values",
	RunE: func(cmd *cobra.Command, args []string) error {
		config := map[string]interface{}{
			"provisionAcrSubscriptionID": "",
			"provisionAcrResourceGroup":  "",
			"provisionAcrRegistryName":   "",
			"provisionEcrRegion":         "",
			"provisionEcrRepository":     "",
			"provisionGcrProjectID":      "",
			"docker_username":            "",
			"docker_password":            "",
			"sourceTag":                  "",
			"targetTag":                  "",
			"namespace":                  "",
			"chartName":                  "",
			"releaseName":                "",
		}

		file, err := os.Create("smurf.yaml")
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