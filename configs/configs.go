package configs

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v2"
)

var BuildKit bool

// Config represents the structure of the configuration file
func LoadConfig(filePath string) (*Config, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			cwd, cwdErr := os.Getwd()
			if cwdErr != nil {
				cwd = filePath
			}
			return nil, fmt.Errorf("no %s found in %s. Pass flags explicitly or run 'smurf init' to create one", FileName, cwd)
		}
		return nil, fmt.Errorf("unable to read the file %v", err)
	}

	var config Config
	err = yaml.Unmarshal(data, &config)
	if err != nil {
		return nil, fmt.Errorf("unable to unmarshal the yaml file %v", err)
	}

	expandConfigEnv(&config)

	return &config, nil
}

// expandConfigEnv expands ${VAR} / $VAR references in every string field of the
// Config struct via os.ExpandEnv, so users can reference environment variables
// (e.g. docker_password: ${DOCKER_PASSWORD}) instead of writing plaintext secrets.
// Missing environment variables expand to an empty string, matching os.ExpandEnv's
// default behavior.
func expandConfigEnv(config *Config) {
	config.Sdkr.DockerPassword = os.ExpandEnv(config.Sdkr.DockerPassword)
	config.Sdkr.DockerUsername = os.ExpandEnv(config.Sdkr.DockerUsername)
	config.Sdkr.GithubUsername = os.ExpandEnv(config.Sdkr.GithubUsername)
	config.Sdkr.GithubToken = os.ExpandEnv(config.Sdkr.GithubToken)
	config.Sdkr.ProvisionAcrRegistryName = os.ExpandEnv(config.Sdkr.ProvisionAcrRegistryName)
	config.Sdkr.ProvisionAcrResourceGroup = os.ExpandEnv(config.Sdkr.ProvisionAcrResourceGroup)
	config.Sdkr.ProvisionAcrSubscriptionID = os.ExpandEnv(config.Sdkr.ProvisionAcrSubscriptionID)
	config.Sdkr.ProvisionGcrProjectID = os.ExpandEnv(config.Sdkr.ProvisionGcrProjectID)
	config.Sdkr.GoogleApplicationCredentials = os.ExpandEnv(config.Sdkr.GoogleApplicationCredentials)
	config.Sdkr.ImageName = os.ExpandEnv(config.Sdkr.ImageName)
	config.Sdkr.TargetImageTag = os.ExpandEnv(config.Sdkr.TargetImageTag)
	config.Sdkr.AwsAccessKey = os.ExpandEnv(config.Sdkr.AwsAccessKey)
	config.Sdkr.AwsSecretKey = os.ExpandEnv(config.Sdkr.AwsSecretKey)
	config.Sdkr.AwsRegion = os.ExpandEnv(config.Sdkr.AwsRegion)
	config.Sdkr.Dockerfile = os.ExpandEnv(config.Sdkr.Dockerfile)

	config.Selm.ReleaseName = os.ExpandEnv(config.Selm.ReleaseName)
	config.Selm.Namespace = os.ExpandEnv(config.Selm.Namespace)
	config.Selm.ChartName = os.ExpandEnv(config.Selm.ChartName)
	config.Selm.FileName = os.ExpandEnv(config.Selm.FileName)
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
			return fmt.Errorf("error setting variable %v: %v", key, err)
		}
	}

	return nil
}
