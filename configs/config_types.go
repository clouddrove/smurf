package configs

var FileName = "smurf.yaml"

type Config struct {
	Sdkr SdkrConfig `yaml:"sdkr"`
	Selm SelmConfig `yaml:"selm"`
}

type SdkrConfig struct {
	DockerPassword               string `yaml:"docker_password"`
	DockerUsername               string `yaml:"docker_username"`
	ProvisionAcrRegistryName     string `yaml:"provisionAcrRegistryName"`
	ProvisionAcrResourceGroup    string `yaml:"provisionAcrResourceGroup"`
	ProvisionAcrSubscriptionID   string `yaml:"provisionAcrSubscriptionID"`
	ProvisionEcrRegion           string `yaml:"provisionEcrRegion"`
	ProvisionEcrRepository       string `yaml:"provisionEcrRepository"`
	ProvisionGcrProjectID        string `yaml:"provisionGcrProjectID"`
	GoogleApplicationCredentials string `yaml:"google_application_credentials"`
	SourceTag                    string `yaml:"sourceTag"`
	TargetTag                    string `yaml:"targetTag"`
}

type SelmConfig struct {
	ReleaseName string `yaml:"releaseName"`
	Namespace   string `yaml:"namespace"`
	ChartName   string `yaml:"chartName"`
	Revision    int    `yaml:"revision"`
}
