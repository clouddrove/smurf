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
		fmt.Printf("Installing Helm plugin: %s\n", plugin)
		cmd := exec.Command("helm", "plugin", "install", plugin)
		output, err := cmd.CombinedOutput()
		if err != nil {
			fmt.Printf("Failed to install Helm plugin: %s\nError: %v\nOutput: %s\n", plugin, err, output)
			continue
		}
		fmt.Printf("Helm plugin installed successfully: %s\n", plugin)
	}
	return nil
}