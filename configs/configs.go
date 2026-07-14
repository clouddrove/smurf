package configs

import (
	"fmt"
	"os"
	"regexp"

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

// bracedEnvVarPattern matches only the braced form ${VAR}. The bare $VAR form is
// deliberately not supported: credentials often contain literal $ characters
// (e.g. P@ss$word123), and os.ExpandEnv-style bare expansion would silently drop
// them as references to unset variables.
var bracedEnvVarPattern = regexp.MustCompile(`\$\{[A-Za-z_][A-Za-z0-9_]*\}`)

// expandBracedEnv expands ${VAR} references in s from the environment, leaving
// every other $ untouched (including bare $VAR). Missing environment variables
// expand to an empty string.
func expandBracedEnv(s string) string {
	return bracedEnvVarPattern.ReplaceAllStringFunc(s, func(match string) string {
		name := match[2 : len(match)-1] // strip ${ and }
		return os.Getenv(name)
	})
}

// expandConfigEnv expands ${VAR} references in every string field of the Config
// struct, so users can reference environment variables
// (e.g. docker_password: ${DOCKER_PASSWORD}) instead of writing plaintext secrets.
// Only the braced ${VAR} form is interpolated; bare $VAR and any other literal $
// are preserved as-is. Missing environment variables expand to an empty string.
func expandConfigEnv(config *Config) {
	config.Sdkr.DockerPassword = expandBracedEnv(config.Sdkr.DockerPassword)
	config.Sdkr.DockerUsername = expandBracedEnv(config.Sdkr.DockerUsername)
	config.Sdkr.GithubUsername = expandBracedEnv(config.Sdkr.GithubUsername)
	config.Sdkr.GithubToken = expandBracedEnv(config.Sdkr.GithubToken)
	config.Sdkr.ProvisionAcrRegistryName = expandBracedEnv(config.Sdkr.ProvisionAcrRegistryName)
	config.Sdkr.ProvisionAcrResourceGroup = expandBracedEnv(config.Sdkr.ProvisionAcrResourceGroup)
	config.Sdkr.ProvisionAcrSubscriptionID = expandBracedEnv(config.Sdkr.ProvisionAcrSubscriptionID)
	config.Sdkr.ProvisionGcrProjectID = expandBracedEnv(config.Sdkr.ProvisionGcrProjectID)
	config.Sdkr.GoogleApplicationCredentials = expandBracedEnv(config.Sdkr.GoogleApplicationCredentials)
	config.Sdkr.ImageName = expandBracedEnv(config.Sdkr.ImageName)
	config.Sdkr.TargetImageTag = expandBracedEnv(config.Sdkr.TargetImageTag)
	config.Sdkr.AwsAccessKey = expandBracedEnv(config.Sdkr.AwsAccessKey)
	config.Sdkr.AwsSecretKey = expandBracedEnv(config.Sdkr.AwsSecretKey)
	config.Sdkr.AwsRegion = expandBracedEnv(config.Sdkr.AwsRegion)
	config.Sdkr.Dockerfile = expandBracedEnv(config.Sdkr.Dockerfile)

	config.Selm.ReleaseName = expandBracedEnv(config.Selm.ReleaseName)
	config.Selm.Namespace = expandBracedEnv(config.Selm.Namespace)
	config.Selm.ChartName = expandBracedEnv(config.Selm.ChartName)
	config.Selm.FileName = expandBracedEnv(config.Selm.FileName)
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
