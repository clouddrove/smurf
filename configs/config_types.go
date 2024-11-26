package configs

var FileName = "smurf.yaml"

type Config struct {
	ProvisionAcrSubscriptionID   string `yaml:"provisionAcrSubscriptionID"`
	ProvisionAcrResourceGroup    string `yaml:"provisionAcrResourceGroup"`
	ProvisionAcrRegistryName     string `yaml:"provisionAcrRegistryName"`
	ProvisionEcrRegion           string `yaml:"provisionEcrRegion"`
	ProvisionEcrRepository       string `yaml:"provisionEcrRepository"`
	ProvisionGcrProjectID        string `yaml:"provisionGcrProjectID"`
	DockerUsername               string `yaml:"docker_username"`
	DockerPassword               string `yaml:"docker_password"`
	SourceTag                    string `yaml:"sourceTag"`
	TargetTag                    string `yaml:"targetTag"`
	Namespace                    string `yaml:"namespace"`
	Revision                     int    `yaml:"revision"`
	ChartName                    string `yaml:"chartName"`
	ChartDir                     string `yaml:"chartDir"`
	GoogleApplicationCredentials string `yaml:"google_application_credentials"`
}
