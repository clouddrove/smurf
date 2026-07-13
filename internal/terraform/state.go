package terraform

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"sort"
	"strings"

	"github.com/clouddrove/smurf/internal/ai"
	"github.com/clouddrove/smurf/internal/utils"
	"github.com/hashicorp/terraform-exec/tfexec"
	tfjson "github.com/hashicorp/terraform-json"
	"github.com/pterm/pterm"
)

// StateList lists all Terraform resources currently tracked in the state file.
//
// format selects the output shape: "table" (default) keeps the existing
// human-facing lines; "json" prints the resource addresses as a single JSON
// array to stdout and suppresses every other stdout write (progress
// messages, AI explanations), so pipelines consuming stdout only ever see
// that array.
func StateList(dir, format string, useAI bool) error {
	isTable := format == "" || format == "table"

	if !isTable {
		// GetTerraform prints via pterm on failure (e.g. "terraform binary
		// not found"); route that to stderr so stdout stays JSON-only. Safe
		// as a one-way switch here: each smurf invocation is a short-lived
		// process handling exactly one command.
		pterm.SetDefaultOutput(os.Stderr)
	}

	tf, err := GetTerraform(dir)
	if err != nil {
		if isTable {
			Error("Failed to initialize Terraform: %v", err)
			ai.AIExplainError(useAI, err.Error())
		}
		return err
	}

	state, err := tf.Show(context.Background())
	if err != nil {
		if isTable {
			Error("Unable to read Terraform state: %v", err)
			ai.AIExplainError(useAI, err.Error())
		}
		return fmt.Errorf("failed to read state: %v", err)
	}

	var resources []string
	if state != nil && state.Values != nil && state.Values.RootModule != nil {
		resources = getAllResources(state.Values.RootModule)
		sort.Strings(resources)
	}

	if !isTable {
		return utils.PrintJSON(resourceAddressesForJSON(resources))
	}

	// No resources found
	if len(resources) == 0 {
		Warn("No resources found in the current Terraform state.")
		return nil
	}

	Info("Resources found in Terraform state:")
	for _, addr := range resources {
		fmt.Printf("  %s\n", addr)
	}

	Success("Total %d resources listed.", len(resources))
	return nil
}

// resourceAddressesForJSON returns resources as a non-nil slice so the JSON
// array is always "[]" rather than "null" when the state has no resources.
func resourceAddressesForJSON(resources []string) []string {
	if resources == nil {
		return []string{}
	}
	return resources
}

// StateResourceAddresses returns the addresses of resources currently
// tracked in the Terraform state, for use in shell completion. Unlike
// StateList it never prints and takes a context: the underlying `terraform
// show` process is killed as soon as ctx is done, so a slow or unreachable
// backend can't hang shell completion. Callers should pass a context with a
// short (2-3s) timeout.
//
// This does not run `terraform init`; if the working directory has not been
// initialized, Show simply fails fast with an error (no side effects), which
// the caller should treat as "no completions available".
//
// It intentionally builds its own *tfexec.Terraform instead of calling
// GetTerraform: that helper prints an error message as a side effect when
// the terraform binary is missing or the instance can't be created, which
// would violate the "completion functions never print" rule.
func StateResourceAddresses(ctx context.Context, dir string) ([]string, error) {
	terraformBinary, err := exec.LookPath("terraform")
	if err != nil {
		return nil, err
	}

	workingDir := "."
	if dir != "" {
		workingDir = dir
	}

	tf, err := tfexec.NewTerraform(workingDir, terraformBinary)
	if err != nil {
		return nil, err
	}

	state, err := tf.Show(ctx)
	if err != nil {
		return nil, err
	}

	if state == nil || state.Values == nil || state.Values.RootModule == nil {
		return nil, nil
	}

	resources := getAllResources(state.Values.RootModule)
	sort.Strings(resources)
	return resources, nil
}

// getAllResources recursively collects resources from all modules
func getAllResources(module *tfjson.StateModule) []string {
	if module == nil {
		return nil
	}

	var addresses []string
	for _, resource := range module.Resources {
		// Skip data sources, only show managed resources
		if !strings.HasPrefix(resource.Type, "data.") {
			addresses = append(addresses, resource.Address)
		}
	}

	for _, childModule := range module.ChildModules {
		addresses = append(addresses, getAllResources(childModule)...)
	}

	return addresses
}

// ErrorHandler provides consistent CLI error output across Terraform operations.
func ErrorHandler(err error) {
	if err != nil {
		Error("Command failed: %v", err)
	}
}
