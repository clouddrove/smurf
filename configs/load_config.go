package configs

import (
	"fmt"
	"io/ioutil"
	"os"

	"gopkg.in/yaml.v2"
)

func LoadConfig(filePath string) (*Config, error) {
	data, err := ioutil.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	var config Config
	err = yaml.Unmarshal(data, &config)
	if err != nil {
		return nil, err
	}

	return &config, nil
}

func setEnvironmentVariable(key, value string) error {
	return os.Setenv(key, value)
}

func ExportEnvironmentVariables(vars map[string]string) error {
	for key, value := range vars {
		if err := setEnvironmentVariable(key, value); err != nil {
			return fmt.Errorf("error setting variable %s: %v", key, err)
		}
	}
	return nil
}
