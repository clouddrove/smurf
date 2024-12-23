package configs

import (
	"bytes"
	"io"
	"time"
)

// types for SDKR
var (
	DockerfilePath   string
	NoCache          bool
	BuildArgs        []string
	Target           string
	Platform         string
	ContextDir       string
	BuildTimeout     time.Duration
	SubscriptionID   string
	ResourceGroup    string
	RegistryName     string
	SarifFile        string
	ConfirmAfterPush bool
	DeleteAfterPush  bool
	ProjectID        string
	Region           string
	Repository       string
)

// types for SELM
var (
	Directory string
	File      []string
	Namespace string
	Timeout   int
	Atomic    bool
	Debug     bool
	Set       []string
	Force     bool
	Wait      bool
)

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
	ImageName                    string `yaml:"imageName"`
	TargetImageTag               string `yaml:"targetImageTag"`
}

type SelmConfig struct {
	ReleaseName string `yaml:"releaseName"`
	Namespace   string `yaml:"namespace"`
	ChartName   string `yaml:"chartName"`
	Revision    int    `yaml:"revision"`
}

type CustomColorWriter struct {
	Buffer *bytes.Buffer
	Writer io.Writer
}
