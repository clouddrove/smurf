package helm_test

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/clouddrove/smurf/internal/helm"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"helm.sh/helm/v3/pkg/repo"
)

func TestHelmInstall(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "helm-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	chartDir := filepath.Join(tmpDir, "testchart")
	err = os.MkdirAll(chartDir, 0755)
	require.NoError(t, err)

	chartYaml := `
apiVersion: v2
name: testchart
description: A test chart
version: 0.1.0
`
	err = os.WriteFile(filepath.Join(chartDir, "Chart.yaml"), []byte(chartYaml), 0644)
	require.NoError(t, err)

	valuesYaml := `
replicaCount: 1
image:
  repository: nginx
  tag: latest
`
	valuesFile := filepath.Join(tmpDir, "values.yaml")
	err = os.WriteFile(valuesFile, []byte(valuesYaml), 0644)
	require.NoError(t, err)

	repoConfig := filepath.Join(tmpDir, "repositories.yaml")
	repoCache := filepath.Join(tmpDir, "repository-cache")
	err = os.MkdirAll(repoCache, 0755)
	require.NoError(t, err)

	os.Setenv("HELM_REPOSITORY_CONFIG", repoConfig)
	os.Setenv("HELM_REPOSITORY_CACHE", repoCache)

	repoFile := repo.NewFile()
	repoFile.Add(&repo.Entry{
		Name: "stable",
		URL:  "https://charts.helm.sh/stable",
	})
	err = repoFile.WriteFile(repoConfig, 0644)
	require.NoError(t, err)

	tests := []struct {
		name        string
		releaseName string
		chartRef    string
		namespace   string
		valuesFiles []string
		duration    time.Duration
		atomic      bool
		debug       bool
		setValues   []string
		repoURL     string
		version     string
		wantErr     bool
		wait        bool
	}{
		{
			name:        "Local chart installation",
			releaseName: "test-release-install",
			chartRef:    chartDir,
			namespace:   "test-namespace",
			valuesFiles: []string{valuesFile},
			duration:    time.Minute * 5,
			atomic:      true,
			debug:       true,
			setValues:   []string{"replicaCount=2"},
			wantErr:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := helm.HelmInstall(
				tt.releaseName,
				tt.chartRef,
				tt.namespace,
				tt.valuesFiles,
				tt.duration,
				tt.atomic,
				tt.debug,
				tt.setValues,
				[]string{},
				tt.repoURL,
				tt.version,
				tt.wait,
			)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
