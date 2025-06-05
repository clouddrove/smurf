package helm

import (
	"os/exec"
	"strings"

	"github.com/pterm/pterm"
)

// HelmPluginInstall installs one or more Helm plugins
func HelmPluginInstall(plugins string) error {
	pluginList := strings.Split(plugins, ",")
	for _, plugin := range pluginList {
		plugin = strings.TrimSpace(plugin)
		if plugin == "" {
			pterm.Warning.Printfln("Helm plugin empty")
			continue
		}

		// Check if plugin is already installed
		checkCmd := exec.Command("helm", "plugin", "list")
		output, err := checkCmd.CombinedOutput()
		if err != nil {
			pterm.Warning.Printfln("Failed to check existing Helm plugins: %v\n", err)
			continue
		}

		if strings.Contains(string(output), plugin) {
			pterm.Warning.Printfln("Helm plugin already installed: %s\n", plugin)
			continue
		}

		// Install plugin if not found
		pterm.Info.Printfln("Installing Helm plugin: %s\n", plugin)
		cmd := exec.Command("helm", "plugin", "install", plugin)
		installOutput, err := cmd.CombinedOutput()
		if err != nil {
			pterm.Warning.Printfln("Failed to install Helm plugin: %s\nError: %v\nOutput: %s\n", plugin, err, installOutput)
			continue
		}
		pterm.Success.Printfln("Helm plugin installed successfully: %s\n", plugin)
	}
	return nil
}
