package helm

import (
	"os"
	"path/filepath"

	"k8s.io/client-go/util/homedir"
)

// settings holds the path to the kubeconfig file.
func init() {
	if os.Getenv("KUBECONFIG") != "" {
		settings.KubeConfig = os.Getenv("KUBECONFIG")
	} else {
		home := homedir.HomeDir()
		settings.KubeConfig = filepath.Join(home, ".kube", "config")
	}
}