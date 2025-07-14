package helm

import (
	"helm.sh/helm/v3/pkg/cli"
	"k8s.io/client-go/kubernetes"
)

// settings holds the path to the kubeconfig file.
// kubeClientset is the Kubernetes clientset.
var (
	settings      = cli.New()
	kubeClientset *kubernetes.Clientset
)

// Resource represents a Kubernetes resource.
type Resource struct {
	Kind      string
	Name      string
	Namespace string
}

// RollbackOptions holds the options for rolling back a Helm release.
type RollbackOptions struct {
	Namespace string
	Debug     bool
	Force     bool
	Timeout   int
	Wait      bool
}
