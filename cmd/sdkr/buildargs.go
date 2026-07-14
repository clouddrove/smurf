package sdkr

import "github.com/clouddrove/smurf/configs"

func sdkrBuildArgs() (map[string]string, error) {
	return configs.ParseCLIBuildArgs(configs.BuildArgs)
}

func buildArgsFrom(args []string) (map[string]string, error) {
	return configs.ParseCLIBuildArgs(args)
}
