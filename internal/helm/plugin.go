package helm

import (
	"fmt"
	"os/exec"
	"strings"
)

// HelmPluginInstall installs one or more Helm plugins
func HelmPluginInstall(plugins string) error {
	pluginList := strings.Split(plugins, ",")
	for _, plugin := range pluginList {
		plugin = strings.TrimSpace(plugin)
		if plugin == "" {
			continue
		}

		// Check if plugin is already installed
		checkCmd := exec.Command("helm", "plugin", "list")
		output, err := checkCmd.CombinedOutput()
		if err != nil {
			fmt.Printf("Failed to check existing Helm plugins: %v\n", err)
			continue
		}

		if strings.Contains(string(output), plugin) {
			fmt.Printf("Helm plugin already installed: %s\n", plugin)
			continue
		}

		// Install plugin if not found
		fmt.Printf("Installing Helm plugin: %s\n", plugin)
		cmd := exec.Command("helm", "plugin", "install", plugin)
		installOutput, err := cmd.CombinedOutput()
		if err != nil {
			fmt.Printf("Failed to install Helm plugin: %s\nError: %v\nOutput: %s\n", plugin, err, installOutput)
			continue
		}
		fmt.Printf("Helm plugin installed successfully: %s\n", plugin)
	}
	return nil
}
