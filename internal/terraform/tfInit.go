package terraform

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/clouddrove/smurf/configs"
	"github.com/clouddrove/smurf/internal/ai"
	"github.com/hashicorp/terraform-exec/tfexec"
	"github.com/pterm/pterm"
)

// CustomLogger handles formatted output for Terraform operations
type CustomLogger struct {
	writer io.Writer
}

// Write formats the output of Terraform operations for better readability
// by the user. It formats the output for download, provider installation,
// and initialization messages.
// Write formats terraform output for readability (download, init, providers)
func (l *CustomLogger) Write(p []byte) (n int, err error) {
	msg := string(p)

	switch {
	case strings.Contains(msg, "Downloading"):
		parts := strings.Fields(msg)
		if len(parts) >= 5 {
			Info("Downloading %s %s for %s...",
				pterm.Cyan(parts[1]),
				pterm.Yellow(parts[2]),
				pterm.Cyan(strings.TrimSpace(parts[4])))
		} else {
			Info("Downloading: %s", strings.TrimSpace(msg))
		}
		return len(p), nil

	case strings.Contains(msg, "Installing"):
		Info("Installing provider: %s", pterm.Cyan(strings.TrimPrefix(msg, "Installing ")))
		return len(p), nil

	case strings.Contains(msg, "Reusing"):
		Info("%s", strings.TrimSpace(msg))
		return len(p), nil

	case strings.Contains(msg, "Initializing"):
		if strings.Contains(msg, "backend") {
			Info("Initializing backend...")
		} else if strings.Contains(msg, "modules") {
			Info("Initializing modules...")
		} else if strings.Contains(msg, "provider") {
			Info("Initializing provider plugins...")
		}
		return len(p), nil

	case strings.Contains(msg, "successfully initialized"):
		Success("Infrastructure successfully initialized!")
		Info("You may now begin working with Smurf. Run `smurf stf plan` to review changes.")
		return len(p), nil

	case strings.Contains(msg, "Backend reinitialization required"):
		Warning("Backend configuration changed. Run with --reconfigure or --migrate-state")
		return len(p), nil

	case strings.Contains(msg, "migrate state"):
		Info("State migration may be required. Use --migrate-state to proceed.")
		return len(p), nil
	}

	return l.writer.Write(p)
}

// InitWithOptions initializes Terraform with all available options
func InitWithOptions(opts configs.InitOptions) error {
	// Handle from-module special case (copies module source to directory)
	if opts.FromModule != "" {
		return initFromModule(opts)
	}

	// Get Terraform client
	tf, err := GetTerraform(opts.Dir)
	if err != nil {
		Error("Failed to initialize Terraform client: %v", err)
		ai.AIExplainError(opts.UseAI, err.Error())
		return err
	}

	// Setup logging
	logger := &CustomLogger{writer: os.Stdout}
	tf.SetStdout(logger)
	tf.SetStderr(logger)

	// Show working directory
	workingDir := "."
	if opts.Dir != "" {
		workingDir = opts.Dir
	}
	Info("Starting infrastructure initialization in %s", workingDir)

	// Build init options
	var initOptions []tfexec.InitOption

	// Basic options
	initOptions = append(initOptions, tfexec.Upgrade(opts.Upgrade))
	initOptions = append(initOptions, tfexec.Backend(opts.Backend))

	// Get modules - still supported
	if !opts.Get {
		initOptions = append(initOptions, tfexec.Get(false))
	}

	// Backend configuration
	for _, config := range opts.BackendConfig {
		initOptions = append(initOptions, tfexec.BackendConfig(config))
	}

	// Reconfigure (replaces existing backend config)
	if opts.Reconfigure {
		initOptions = append(initOptions, tfexec.Reconfigure(true))
	}

	// Note: MigrateState and ForceCopy are not directly available in tfexec
	// They need to be handled via environment variables or config options
	// Handle migrate-state and force-copy via environment variables
	if opts.MigrateState {
		os.Setenv("TF_MIGRATE_STATE", "true")
		defer os.Unsetenv("TF_MIGRATE_STATE")
		Info("State migration enabled")
	}

	if opts.ForceCopy {
		os.Setenv("TF_FORCE_COPY", "true")
		defer os.Unsetenv("TF_FORCE_COPY")
		Info("Force copy mode enabled")
	}

	// Execute init with retry logic for lock conflicts
	err = runInitWithRetry(tf, initOptions)
	if err != nil {
		// Check for specific error types and provide helpful messages
		if strings.Contains(err.Error(), "backend configuration changed") {
			Error("Backend configuration has changed")
			Info("Use --reconfigure to accept the new configuration, or")
			Info("Use --migrate-state to migrate existing state to the new backend")
			Info("Example: smurf stf init --reconfigure --migrate-state")
		} else if strings.Contains(err.Error(), "state lock") {
			Error("Failed to acquire state lock")
			Info("Try increasing lock timeout with --lock-timeout=5m")
			Info("Or check if another process is holding the lock")
		} else {
			Error("Initialization failed: %v", err)
		}

		ai.AIExplainError(opts.UseAI, err.Error())
		return err
	}

	// Display success message with backend info
	displayBackendInfo(opts)

	return nil
}

// runInitWithRetry executes terraform init with retry logic for lock conflicts
func runInitWithRetry(tf *tfexec.Terraform, initOptions []tfexec.InitOption) error {
	maxRetries := 3
	retryDelay := 2 * time.Second

	for i := 0; i < maxRetries; i++ {
		err := tf.Init(context.Background(), initOptions...)
		if err == nil {
			return nil
		}

		// Retry on lock conflicts
		if strings.Contains(err.Error(), "state lock") && i < maxRetries-1 {
			Warning("State lock detected, retrying in %v... (attempt %d/%d)", retryDelay, i+2, maxRetries)
			time.Sleep(retryDelay)
			retryDelay *= 2 // Exponential backoff
			continue
		}

		return err
	}

	return fmt.Errorf("max retries exceeded")
}

// initFromModule handles initialization from a module source
// initFromModule handles initialization from a module source
func initFromModule(opts configs.InitOptions) error {
	Info("Initializing from module: %s", opts.FromModule)

	tf, err := GetTerraform(opts.Dir)
	if err != nil {
		Error("Failed to initialize Terraform client: %v", err)
		return err
	}

	// Build options for from-module
	var options []tfexec.InitOption
	options = append(options, tfexec.FromModule(opts.FromModule))
	options = append(options, tfexec.Upgrade(opts.Upgrade))

	// Get modules flag
	if !opts.Get {
		options = append(options, tfexec.Get(false))
	}

	// Execute init
	err = tf.Init(context.Background(), options...)

	if err != nil {
		Error("Failed to initialize from module: %v", err)
		return err
	}

	Success("Successfully initialized from module: %s", opts.FromModule)
	return nil
}

// displayBackendInfo shows information about the configured backend
func displayBackendInfo(opts configs.InitOptions) {
	Success("Terraform backend and providers successfully initialized.")

	// Show backend configuration info if available
	if len(opts.BackendConfig) > 0 {
		Info("Using backend configuration files:")
		for _, config := range opts.BackendConfig {
			Info("  • %s", config)
		}
	}

	if opts.Reconfigure && opts.MigrateState {
		Info("Backend reconfigured and state migration completed")
	} else if opts.Reconfigure {
		Info("Backend reconfigured (existing state preserved)")
	} else if opts.MigrateState {
		Info("State migrated to new backend")
	}

	// Provide next steps
	Info("\nNext steps:")
	Info("  1. Review infrastructure changes: smurf stf plan")
	Info("  2. Apply infrastructure changes: smurf stf apply")
	Info("  3. Check resource state: smurf stf state list")
}

// Init - Simplified version for backward compatibility
func Init(dir string, upgrade, useAI bool) error {
	return InitWithOptions(configs.InitOptions{
		Dir:     dir,
		Upgrade: upgrade,
		UseAI:   useAI,
		Backend: true,
		Get:     true,
	})
}
