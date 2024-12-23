package helm

import (
	"os"
	"path/filepath"

	"k8s.io/client-go/util/homedir"
)



func init() {
	if os.Getenv("KUBECONFIG") != "" {
		settings.KubeConfig = os.Getenv("KUBECONFIG")
	} else {
		home := homedir.HomeDir()
		settings.KubeConfig = filepath.Join(home, ".kube", "config")
	}
}