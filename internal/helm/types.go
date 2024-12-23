package helm

import (
	"helm.sh/helm/v3/pkg/cli"
	"k8s.io/client-go/kubernetes"
)

var (
	settings      = cli.New()
	kubeClientset *kubernetes.Clientset
)

type Resource struct {
	Kind string
	Name string
}

type RollbackOptions struct {
	Namespace string
	Debug     bool
	Force     bool
	Timeout   int
	Wait      bool
}
