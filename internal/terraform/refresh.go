package terraform

import (
	"context"
	"fmt"
	"time"

	"github.com/hashicorp/terraform-exec/tfexec"
	tfjson "github.com/hashicorp/terraform-json"
	"github.com/pterm/pterm"
)

// Refresh updates the state file of your infrastructure
func Refresh(vars []string, varFiles []string, lock bool, dir string) error {
	tf, err := GetTerraform(dir)
	if err != nil {
		return err
	}

	spinner, _ := pterm.DefaultSpinner.Start("Refreshing state...")

	applyOptions := []tfexec.RefreshCmdOption{}

	if vars != nil {
		pterm.Info.Printf("Setting variable: %s\n", vars)
		for _, v := range vars {
			pterm.Info.Printf("Setting variable: %s\n", v)
			applyOptions = append(applyOptions, tfexec.Var(v))
		}
	}

	if varFiles != nil {
		pterm.Info.Printf("Setting variable file: %s\n", varFiles)
		for _, vf := range varFiles {
			pterm.Info.Printf("Loading variable file: %s\n", vf)
			applyOptions = append(applyOptions, tfexec.VarFile(vf))
		}
	}

	if !lock {
		fmt.Printf("%s\n\n", pterm.Yellow("Note: State locking is disabled!"))
	}

	startTime := time.Now()

	currentState, err := tf.Show(context.Background())
	if err == nil && currentState != nil && currentState.Values != nil {
		spinner.Stop()

		resources := getAllRefreshResources(currentState.Values.RootModule)
		for _, resource := range resources {
			fmt.Printf("%s: Refreshing state... [id=%s]\n",
				resource.Address,
				resource.AttributeValues["id"].(string))
		}
		spinner, _ = pterm.DefaultSpinner.Start("Refreshing state...")
	}

	err = tf.Refresh(context.Background(), applyOptions...)
	if err != nil {
		fmt.Printf("%s Error refreshing state: %v\n\n", pterm.Red("│"), err)
		fmt.Printf("%s Terraform failed to refresh state. The error shown above occurred\n", pterm.Red("│"))
		fmt.Printf("%s when Terraform attempted to refresh the state file.\n\n", pterm.Red("│"))
		spinner.Fail("Failed to refresh state")
		return err
	}

	state, err := tf.Show(context.Background())
	if err != nil {
		spinner.Fail("Failed to read updated state")
		return fmt.Errorf("error reading updated state: %v", err)
	}

	fmt.Printf("\nRefresh complete! Resources: %d refreshed.\n", countResources(state.Values.RootModule))
	spinner.Success("Refreshed successfully")

	duration := time.Since(startTime).Round(time.Second)
	fmt.Printf("Execute time: %s\n", duration)

	return nil
}

// getAllResources recursively collects all resources from all modules
func getAllRefreshResources(module *tfjson.StateModule) []*tfjson.StateResource {
	if module == nil {
		return nil
	}

	resources := module.Resources

	for _, childModule := range module.ChildModules {
		resources = append(resources, getAllRefreshResources(childModule)...)
	}

	return resources
}

// countResources recursively counts resources in all modules
func countResources(module *tfjson.StateModule) int {
	if module == nil {
		return 0
	}

	count := len(module.Resources)
	for _, childModule := range module.ChildModules {
		count += countResources(childModule)
	}
	return count
}
