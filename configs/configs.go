package configs

import (
	"fmt"
	"os"

	"github.com/pterm/pterm"
	"gopkg.in/yaml.v2"
)

var BuildKit bool

// Config represents the structure of the configuration file
func LoadConfig(filePath string) (*Config, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		pterm.Error.Printfln("Unable to read the file %v", err)
		return nil, fmt.Errorf("unable to read the file %v", err)
	}

	var config Config
	err = yaml.Unmarshal(data, &config)
	if err != nil {
		pterm.Error.Printfln("Unable to unmarshal the yaml file %v", err)
		return nil, fmt.Errorf("unable to unmarshal the yaml file %v", err)
	}

	pterm.Success.Printfln("Successfuly load the config...")
	return &config, nil
}

// Set the Environment Variable for the usage in the internal functions
// used for credential management
func setEnvironmentVariable(key, value string) error {
	return os.Setenv(key, value)
}

// ExportEnvironmentVariables sets the environment variables for the given map
func ExportEnvironmentVariables(vars map[string]string) error {
	for key, value := range vars {
		if err := setEnvironmentVariable(key, value); err != nil {
			pterm.Error.Printfln("error setting variable %v: %v", key, err)
			return fmt.Errorf("error setting variable %v: %v", key, err)
		}
	}

	pterm.Info.Printfln("Succesfully export the environment variables...")
	return nil
}
