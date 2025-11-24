package configs

// types for SDKR
var (
	DockerfilePath   string
	NoCache          bool
	BuildArgs        []string
	Target           string
	Platform         string
	ContextDir       string
	BuildTimeout     int
	SubscriptionID   string
	ResourceGroup    string
	RegistryName     string
	SarifFile        string
	ConfirmAfterPush bool
	DeleteAfterPush  bool
	ProjectID        string
	Region           string
	Repository       string
	UseGCR           bool
)

// types for SELM
var (
	Directory       string
	File            []string
	Namespace       string
	Timeout         int
	Atomic          bool
	Debug           bool
	Set             []string
	SetLiteral      []string
	Force           bool
	Wait            bool
	CaFile          string // --ca-file
	CertFile        string // --cert-file
	KeyFile         string // --key-file
	Password        string // --password
	RepoURL         string // --repo
	Username        string // --username
	Verify          bool   // --verify
	Version         string // --version
	HelmConfigDir   string
	Destination     string
	Untar           bool
	UntarDir        string
	Keyring         string
	Insecure        bool
	PlainHttp       bool
	PassCredentials bool
	Devel           bool
	Prov            bool
)

// Config struct to hold the configuration for the SDKR and SELM
var FileName = "smurf.yaml"

// Config struct to hold the configuration for the SDKR and SELM
type Config struct {
	Sdkr SdkrConfig `yaml:"sdkr"`
	Selm SelmConfig `yaml:"selm"`
}

// types for SDKR in the config file
type SdkrConfig struct {
	DockerPassword               string `yaml:"docker_password"`
	DockerUsername               string `yaml:"docker_username"`
	GithubUsername               string `yaml:"github_username"`
	GithubToken                  string `yaml:"github_token"`
	ProvisionAcrRegistryName     string `yaml:"provisionAcrRegistryName"`
	ProvisionAcrResourceGroup    string `yaml:"provisionAcrResourceGroup"`
	ProvisionAcrSubscriptionID   string `yaml:"provisionAcrSubscriptionID"`
	ProvisionGcrProjectID        string `yaml:"provisionGcrProjectID"`
	GoogleApplicationCredentials string `yaml:"google_application_credentials"`
	ImageName                    string `yaml:"imageName"`
	TargetImageTag               string `yaml:"targetImageTag"`
	AwsAccessKey                 string `yaml:"awsAccessKey"`
	AwsSecretKey                 string `yaml:"awsSecretKey"`
	AwsRegion                    string `yaml:"awsRegion"`
	Dockerfile                   string `yaml:"dockerfile"`
	AwsECR                       bool   `yaml:"awsECR"`
	DockerHub                    bool   `yaml:"dockerHub"`
	GHCRRepo                     bool   `yaml:"ghcrRepo"`
	GCPRepo                      bool   `yaml:"gcpRepo"`
}

// types for SELM in the config file
type SelmConfig struct {
	HelmDeploy  bool   `yaml:"deployHelm"`
	ReleaseName string `yaml:"releaseName"`
	Namespace   string `yaml:"namespace"`
	ChartName   string `yaml:"chartName"`
	FileName    string `yaml:"fileName"`
	Revision    int    `yaml:"revision"`
}
