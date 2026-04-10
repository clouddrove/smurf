package stf

import (
	"os"

	"github.com/clouddrove/smurf/configs"
	"github.com/clouddrove/smurf/internal/terraform"
	"github.com/spf13/cobra"
)

var (
	initUpgrade       bool
	initDir           string
	initReconfigure   bool
	initMigrateState  bool
	initBackendConfig []string // Support multiple -backend-config flags
	initBackend       bool
	initForceCopy     bool
	initLock          bool
	initLockTimeout   string
	initGet           bool
	initGetPlugins    bool
	initVerifyPlugins bool
	initPluginDir     string
	initFromModule    string
	initRegistryOnly  bool
)

// initCmd defines a subcommand that initializes Terraform.
var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize Terraform",
	Long: `Initialize a Terraform working directory.
		   This command performs several initialization steps including:
        		- Download and install provider plugins
				- Initialize backend configuration
				- Download and install modules
				- Set up workspace configuration`,
	SilenceErrors: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		opts := configs.InitOptions{
			Dir:           initDir,
			Upgrade:       initUpgrade,
			UseAI:         useAI,
			Reconfigure:   initReconfigure,
			MigrateState:  initMigrateState,
			BackendConfig: initBackendConfig,
			Backend:       initBackend,
			ForceCopy:     initForceCopy,
			Lock:          initLock,
			LockTimeout:   initLockTimeout,
			Get:           initGet,
			GetPlugins:    initGetPlugins,
			VerifyPlugins: initVerifyPlugins,
			PluginDir:     initPluginDir,
			FromModule:    initFromModule,
			RegistryOnly:  initRegistryOnly,
		}
		err := terraform.InitWithOptions(opts)
		if err != nil {
			os.Exit(1)
		}
		return nil
	},
	Example: `
 # Basic initialization
  smurf stf init

  # Initialize with backend configuration file
  smurf stf init --backend-config=backend.hcl

  # Initialize with multiple backend config files
  smurf stf init --backend-config=backend.hcl --backend-config=prod-backend.hcl

  # Reconfigure backend (ignore existing config)
  smurf stf init --reconfigure

  # Migrate state to new backend
  smurf stf init --migrate-state

  # Reconfigure and migrate state
  smurf stf init --reconfigure --migrate-state

  # Initialize from module source
  smurf stf init --from-module=github.com/terraform-aws-modules/terraform-aws-vpc

  # Custom plugin directory (via TF_PLUGIN_CACHE_DIR)
  smurf stf init --plugin-dir=/custom/plugins

  # Skip downloading modules
  smurf stf init --get=false
`,
}

func init() {
	// Basic flags
	initCmd.Flags().BoolVar(&initUpgrade, "upgrade", false, "Upgrade installed modules and plugins")
	initCmd.Flags().StringVar(&initDir, "dir", "", "Directory containing Terraform files (default is current directory)")
	initCmd.Flags().BoolVar(&useAI, "ai", false, "Enable AI help mode (requires OPENAI_API_KEY)")

	// Backend configuration flags
	initCmd.Flags().BoolVar(&initReconfigure, "reconfigure", false, "Reconfigure backend, ignoring existing configuration")
	initCmd.Flags().BoolVar(&initMigrateState, "migrate-state", false, "Migrate existing state to new backend")
	initCmd.Flags().StringArrayVar(&initBackendConfig, "backend-config", []string{}, "Path to backend configuration file (can be used multiple times)")
	initCmd.Flags().BoolVar(&initBackend, "backend", true, "Configure backend (disable with --backend=false)")
	initCmd.Flags().BoolVar(&initForceCopy, "force-copy", false, "Suppress prompts about copying state data during backend migration")

	// Module and plugin flags
	initCmd.Flags().BoolVar(&initGet, "get", true, "Download and install modules")
	initCmd.Flags().BoolVar(&initGetPlugins, "get-plugins", true, "Download and install provider plugins")
	initCmd.Flags().BoolVar(&initVerifyPlugins, "verify-plugins", true, "Verify provider plugins with signature checking")
	initCmd.Flags().StringVar(&initPluginDir, "plugin-dir", "", "Directory containing provider plugins")
	initCmd.Flags().StringVar(&initFromModule, "from-module", "", "Copy the source module into the target directory")
	initCmd.Flags().BoolVar(&initRegistryOnly, "registry-only", false, "Only check the official registry for providers")

	// Add command to parent
	stfCmd.AddCommand(initCmd)
}
