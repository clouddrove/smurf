package helm

import (
	"fmt"
	"os/exec"
)

// HelmPluginInstall installs a Helm plugin
func HelmPluginInstall(pluginURL string) error {
	cmd := exec.Command("helm", "plugin", "install", pluginURL)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to install Helm plugin: %v, output: %s", err, output)
	}
	fmt.Printf("Helm plugin installed successfully: %s\n", output)
	return nil
}
