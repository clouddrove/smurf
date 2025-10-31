package cmd

import (
	"fmt"
	"os"

	"github.com/clouddrove/smurf/configs"
	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v2"
)

// generateConfigCmd represents the init command to generate a smurf.yaml configuration file with empty values
var generateConfig = &cobra.Command{
	Use:   "init",
	Short: "Generate a smurf.yaml configuration file with empty values",
	RunE: func(cmd *cobra.Command, args []string) error {
		config := map[string]interface{}{
			"sdkr": map[string]interface{}{
				"awsECR":         false,
				"dockerHub":      false,
				"ghcrRepo":       false,
				"imageName":      "",
				"targetImageTag": "",
				"dockerfile":     "",
			},
			"selm": map[string]interface{}{
				"deployHelm":  false,
				"releaseName": "",
				"namespace":   "",
				"chartName":   "",
				"fileName":    "",
				"revision":    0,
			},
		}

		file, err := os.Create(configs.FileName)
		if err != nil {
			pterm.Error.Printfln("error creating YAML file: %v", err)
			return fmt.Errorf("error creating YAML file: %v", err)
		}
		defer file.Close()

		data, err := yaml.Marshal(&config)
		if err != nil {
			pterm.Error.Printfln("error marshaling data to YAML: %v", err)
			return fmt.Errorf("error marshaling data to YAML: %v", err)
		}

		if _, err := file.Write(data); err != nil {
			pterm.Error.Printfln("error writing to YAML file: %v", err)
			return fmt.Errorf("error writing to YAML file: %v", err)
		}
		fmt.Printf("✅smurf.yaml configuration file generated successfully with empty values.")
		return nil
	},
}

func init() {
	RootCmd.AddCommand(generateConfig)
}
