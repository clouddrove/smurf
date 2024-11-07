package helm

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/fatih/color"
	"github.com/pterm/pterm"
	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chart/loader"
	"helm.sh/helm/v3/pkg/chartutil"
	"helm.sh/helm/v3/pkg/cli"
	"helm.sh/helm/v3/pkg/release"
	"helm.sh/helm/v3/pkg/strvals"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var settings = cli.New()

func init() {
	if os.Getenv("KUBECONFIG") != "" {
		settings.KubeConfig = os.Getenv("KUBECONFIG")
	} else {
		home := homedir.HomeDir()
		settings.KubeConfig = filepath.Join(home, ".kube", "config")
	}
}

func getKubeClient() (*kubernetes.Clientset, error) {
	var kubeconfig string
	if os.Getenv("KUBECONFIG") != "" {
		kubeconfig = os.Getenv("KUBECONFIG")
	} else {
		home := homedir.HomeDir()
		kubeconfig = filepath.Join(home, ".kube", "config")
	}

	config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		return nil, err
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, err
	}

	return clientset, nil
}

func CreateChart(chartName, saveDir string) error {
	spinner, _ := pterm.DefaultSpinner.Start(fmt.Sprintf("Creating chart '%s' in directory '%s'", chartName, saveDir))

	if _, err := os.Stat(saveDir); os.IsNotExist(err) {
		if err := os.MkdirAll(saveDir, os.ModePerm); err != nil {
			spinner.Fail(fmt.Sprintf("Failed to create directory: %s", saveDir))
			color.Red("Error: %v", err)
			return err
		}
	}

	_, err := chartutil.Create(chartName, saveDir)
	if err != nil {
		spinner.Fail(fmt.Sprintf("Failed to create chart '%s'", chartName))
		color.Red("Error: %v", err)
		return err
	}

	spinner.Success(fmt.Sprintf("Chart '%s' created successfully in '%s'", chartName, saveDir))
	color.Green("Chart '%s' has been successfully created in the directory '%s'.", chartName, saveDir)
	return nil
}

func HelmInstall(releaseName, chartPath, namespace string) error {
	settings := cli.New()
	kubeConfigPath := filepath.Join(os.Getenv("HOME"), ".kube", "config")
	settings.KubeConfig = kubeConfigPath

	actionConfig := new(action.Configuration)
	if err := actionConfig.Init(settings.RESTClientGetter(), namespace, os.Getenv("HELM_DRIVER"), func(format string, v ...interface{}) {
		fmt.Printf(format, v...)
	}); err != nil {
		color.Red("Failed to initialize Helm action configuration: %v\n", err)
		return err
	}

	client := action.NewInstall(actionConfig)
	client.ReleaseName = releaseName
	client.Namespace = namespace

	chart, err := loader.Load(chartPath)
	if err != nil {
		color.Red("Failed to load chart: %v\n", err)
		return err
	}

	release, err := client.Run(chart, nil) 
	if err != nil {
		color.Red("Installation failed: %v\n", err)
		return err
	}

	color.Green("NAME: %s\n", release.Name)
	color.Green("LAST DEPLOYED: %s\n", release.Info.LastDeployed)
	color.Green("NAMESPACE: %s\n", release.Namespace)
	color.Green("STATUS: %s\n", release.Info.Status)
	color.Green("REVISION: %d\n", release.Version)
	color.Green("NOTES:\n%s\n", release.Info.Notes)

	color.Cyan("Get the application URL by running these commands:\n")
	color.Cyan("export POD_NAME=$(kubectl get pods --namespace %s -l \"app.kubernetes.io/name=%s,app.kubernetes.io/instance=%s\" -o jsonpath=\"{.items[0].metadata.name}\")\n", namespace, chart.Metadata.Name, release.Name)
	color.Cyan("export CONTAINER_PORT=$(kubectl get pod --namespace %s $POD_NAME -o jsonpath=\"{.spec.containers[0].ports[0].containerPort}\")\n", namespace)
	color.Cyan("echo \"Visit http://127.0.0.1:8080 to use your application\"\n")
	color.Cyan("kubectl --namespace %s port-forward $POD_NAME 8080:$CONTAINER_PORT\n", namespace)

	return nil
}


// ensureNamespace checks and creates the namespace if necessary
func ensureNamespace(namespace string, create bool) error {
    clientset, err := getKubeClient()
    if err != nil {
        return err
    }
    _, err = clientset.CoreV1().Namespaces().Get(context.Background(), namespace, metav1.GetOptions{})
    if err == nil {
        return nil 
    }

    if create {
        ns := &v1.Namespace{
            ObjectMeta: metav1.ObjectMeta{
                Name: namespace,
            },
        }
        _, err = clientset.CoreV1().Namespaces().Create(context.Background(), ns, metav1.CreateOptions{})
        if err != nil {
            return fmt.Errorf("Failed to create namespace '%s': %v", namespace, err)
        }
        fmt.Println("Namespace created successfully")
    } else {
        return fmt.Errorf("Namespace '%s' does not exist and was not created", namespace)
    }

    return nil
}

func HelmUpgrade(releaseName, chartPath, namespace string, setValues []string, valuesFiles []string, createNamespace, atomic bool, timeout time.Duration, debug bool) error {
	settings := cli.New()
	settings.Debug = debug
	spinner, _ := pterm.DefaultSpinner.Start("Upgrading release...")

	if createNamespace {
		if err := ensureNamespace(namespace, true); err != nil {
			spinner.Fail("Failed to ensure namespace: " + err.Error())
			color.Red("Error: %v\n", err)
			return err
		}
	}


	actionConfig := new(action.Configuration)
	if err := actionConfig.Init(settings.RESTClientGetter(), namespace, os.Getenv("HELM_DRIVER"), func(format string, v ...interface{}) {
		fmt.Printf(format, v...)
	}); err != nil {
		spinner.Fail("Failed to initialize Helm action configuration: " + err.Error())
		color.Red("Error: %v\n", err)
		return err
	}

	client := action.NewUpgrade(actionConfig)
	client.Namespace = namespace
	client.Atomic = atomic
	client.Timeout = timeout

	chart, err := loader.Load(chartPath)
	if err != nil {
		spinner.Fail("Failed to load chart: " + err.Error())
		color.Red("Error: %v\n", err)
		return err
	}


	vals := make(map[string]interface{})
	for _, f := range valuesFiles {
		additionalVals, err := chartutil.ReadValuesFile(f)
		if err != nil {
			spinner.Fail(fmt.Sprintf("Failed to read values file: %s", f))
			color.Red("Error reading values file %s: %v\n", f, err)
			return err
		}
		for key, value := range additionalVals {
			vals[key] = value
		}
	}

	for _, set := range setValues {
		if err := strvals.ParseInto(set, vals); err != nil {
			spinner.Fail("Failed to parse set values: " + err.Error())
			color.Red("Error: %v\n", err)
			return err
		}
	}

	rel, err := client.Run(releaseName, chart, vals)
	if err != nil {
		spinner.Fail("Upgrade failed: " + err.Error())
		color.Red("Error: %v\n", err)
		return err
	}

	spinner.Success(fmt.Sprintf("Release '%s' upgraded successfully in namespace '%s'", releaseName, namespace))
	color.Green("NAME: %s\n", rel.Name)
	color.Green("LAST DEPLOYED: %s\n", rel.Info.LastDeployed)
	color.Green("NAMESPACE: %s\n", rel.Namespace)
	color.Green("STATUS: %s\n", rel.Info.Status.String())
	color.Green("REVISION: %d\n", rel.Version)
	color.Green("NOTES:\n%s\n", rel.Info.Notes)

	return nil
}

func HelmList(namespace string) ([]*release.Release, error) {
	settings := cli.New()
	actionConfig := new(action.Configuration)
	spinner, _ := pterm.DefaultSpinner.Start("Listing releases in namespace: " + namespace)

	if err := actionConfig.Init(settings.RESTClientGetter(), namespace, "secrets", nil); err != nil {
		spinner.Fail("Failed to initialize action configuration")
		color.Red("Error: %s", err.Error())
		return nil, err
	}

	client := action.NewList(actionConfig)
	client.AllNamespaces = true

	releases, err := client.Run()
	if err != nil {
		spinner.Fail("Failed to list releases")
		color.Red("Error: %s", err.Error())
		return nil, err
	}

	spinner.Stop()
	fmt.Println()
	color.Cyan("%-17s %-10s %-8s %-20s %-7s %-30s", "NAME", "NAMESPACE", "REVISION", "UPDATED", "STATUS", "CHART")
	for _, rel := range releases {
		updatedStr := rel.Info.LastDeployed.Local().Format("2006-01-02 15:04:05.999999999 -0700 MST")
		color.White("%-17s %-10s %-8d %-20s %-7s %-30s",
			rel.Name, rel.Namespace, rel.Version, updatedStr, rel.Info.Status.String(), rel.Chart.Metadata.Name+"-"+rel.Chart.Metadata.Version)
	}

	return releases, nil
}

func HelmUninstall(releaseName, namespace string) error {
	spinner, _ := pterm.DefaultSpinner.Start(fmt.Sprintf("Uninstalling release '%s'", releaseName))

	actionConfig := new(action.Configuration)
	debugLog := func(format string, v ...interface{}) {
		fmt.Printf(format, v...)
	}
	if err := actionConfig.Init(settings.RESTClientGetter(), namespace, os.Getenv("HELM_DRIVER"), debugLog); err != nil {
		spinner.Fail("Failed to initialize Helm action configuration: " + err.Error())
		return err
	}


	client := action.NewUninstall(actionConfig)
	if client == nil {
		spinner.Fail("Failed to create Helm uninstall client")
		return fmt.Errorf("failed to create Helm uninstall client")
	}

	_, err := client.Run(releaseName)
	if err != nil {
		spinner.Fail("Uninstall failed: " + err.Error())
		return err
	}

	spinner.Success(fmt.Sprintf("Release '%s' uninstalled successfully", releaseName))
	return nil
}

// HelmLint lints a Helm chart
func HelmLint(chartPath string) error {
	spinner, _ := pterm.DefaultSpinner.Start("Linting chart")
	actionConfig := new(action.Configuration)
	_ = actionConfig
	client := action.NewLint()

	result := client.Run([]string{chartPath}, nil)
	if len(result.Messages) > 0 {
		for _, msg := range result.Messages {
			fmt.Println(msg)
		}
		spinner.Fail("Linting issues found")
	} else {
		spinner.Success("No linting issues")
	}
	return nil
}

// HelmTemplate renders the Helm templates for a given chart
func HelmTemplate(releaseName, chartPath, namespace string) error {
	settings := cli.New() 
	actionConfig := new(action.Configuration)

	if err := actionConfig.Init(settings.RESTClientGetter(), namespace, os.Getenv("HELM_DRIVER"), nil); err != nil {
		pterm.DefaultBasicText.WithStyle(pterm.NewStyle(pterm.FgRed)).Println(err.Error())
		return err
	}

	client := action.NewInstall(actionConfig)
	client.DryRun = true 
	client.ReleaseName = releaseName
	client.Namespace = namespace
	client.Replace = true   
	client.ClientOnly = true 


	chart, err := loader.Load(chartPath)
	if err != nil {
		pterm.DefaultBasicText.WithStyle(pterm.NewStyle(pterm.FgRed)).Println(err.Error())
		return err
	}

	spinner, _ := pterm.DefaultSpinner.Start("Rendering Helm templates...")
	rel, err := client.Run(chart, nil) 
	if err != nil {
		spinner.Fail(err.Error())
		return err
	}
	spinner.Success("Templates rendered successfully")

	green := color.New(color.FgGreen).SprintFunc()
	fmt.Println(green(rel.Manifest))

	return nil
}

// HelmProvision provisions a Helm chart by installing or upgrading it, linting it, and rendering its templates
func HelmProvision(releaseName, chartPath, namespace string) error {
	settings := cli.New()
	actionConfig := new(action.Configuration)
	if err := actionConfig.Init(settings.RESTClientGetter(), namespace, os.Getenv("HELM_DRIVER"), nil); err != nil {
		return err
	}

	client := action.NewList(actionConfig)
	results, err := client.Run()
	if err != nil {
		return err
	}

	var wg sync.WaitGroup
	var installErr, upgradeErr, lintErr, templateErr error

	exists := false
	for _, result := range results {
		if result.Name == releaseName {
			exists = true
			break
		}
	}

	wg.Add(1)
	if exists {
		go func() {
			defer wg.Done()
			upgradeErr = HelmUpgrade(releaseName, chartPath, namespace, nil, nil, false, false, 0, false)
		}()
	} else {
		go func() {
			defer wg.Done()
			installErr = HelmInstall(releaseName, chartPath, namespace)
		}()
	}

	wg.Add(2)
	go func() {
		defer wg.Done()
		lintErr = HelmLint(chartPath)
	}()

	go func() {
		defer wg.Done()
		templateErr = HelmTemplate(releaseName, chartPath, namespace)
	}()

	wg.Wait()

	if installErr != nil || upgradeErr != nil || lintErr != nil || templateErr != nil {
		if installErr != nil {
			pterm.Error.Println("Install failed:", installErr)
		}
		if upgradeErr != nil {
			pterm.Error.Println("Upgrade failed:", upgradeErr)
		}
		if lintErr != nil {
			pterm.Error.Println("Lint failed:", lintErr)
		}
		if templateErr != nil {
			pterm.Error.Println("Template rendering failed:", templateErr)
		}
		return fmt.Errorf(color.RedString("Provisioning failed"))
	}

	pterm.Success.Println("Provisioning completed successfully.")
	return nil
}






// HelmReleaseExists checks if a specific release exists in the given namespace
func HelmReleaseExists(releaseName, namespace string) (bool, error) {
    settings := cli.New() 

    if settings.KubeConfig == "" {
        kubeConfigPath := filepath.Join(homedir.HomeDir(), ".kube", "config")
        settings.KubeConfig = kubeConfigPath
    }

    actionConfig := new(action.Configuration)
    if err := actionConfig.Init(settings.RESTClientGetter(), namespace, "secrets", nil); err != nil {
        return false, err
    }

    list := action.NewList(actionConfig)
    list.Deployed = true 
    list.AllNamespaces = false
    releases, err := list.Run()
    if err != nil {
        return false, err
    }

    for _, rel := range releases {
        if rel.Name == releaseName && rel.Namespace == namespace {
            return true, nil 
        }
    }

    return false, nil 
}




// HelmStatus retrieves the status of a Helm release
func HelmStatus(releaseName, namespace string) error {
    settings := cli.New()

    actionConfig := new(action.Configuration)
	if err := actionConfig.Init(settings.RESTClientGetter(), namespace, "secrets", func(format string, v ...interface{}) {
		if settings.Debug {
			fmt.Printf(format, v...)
		}
	}); err != nil {
        return fmt.Errorf("failed to initialize Helm client: %w", err)
    }

    statusAction := action.NewStatus(actionConfig)

    release, err := statusAction.Run(releaseName)
    if err != nil {
        pterm.Error.Println("Retrieving status failed")
        color.Red("Error: %s", err.Error())
        return err
    }

    data := [][]string{
        {"NAME", release.Name},
        {"NAMESPACE", release.Namespace},
        {"STATUS", release.Info.Status.String()},
        {"REVISION", fmt.Sprintf("%d", release.Version)},
        {"TEST SUITE", "None"}, 
    }

    pterm.DefaultTable.WithHasHeader(false).WithData(data).Render()

    if release.Info.Notes != "" {
        pterm.Info.Println("NOTES:")
        fmt.Println(color.CyanString(release.Info.Notes))
        fmt.Println(color.GreenString("Get the application URL by running these commands:"))
        fmt.Println(color.GreenString("export POD_NAME=$(kubectl get pods --namespace " + release.Namespace + " -l \"app.kubernetes.io/name=" + release.Chart.Metadata.Name + ",app.kubernetes.io/instance=" + release.Name + "\" -o jsonpath=\"{.items[0].metadata.name}\")"))
        fmt.Println(color.GreenString("export CONTAINER_PORT=$(kubectl get pod --namespace " + release.Namespace + " $POD_NAME -o jsonpath=\"{.spec.containers[0].ports[0].containerPort}\")"))
        fmt.Println(color.GreenString("echo \"Visit http://127.0.0.1:8080 to use your application\""))
        fmt.Println(color.GreenString("kubectl --namespace " + release.Namespace + " port-forward $POD_NAME 8080:$CONTAINER_PORT"))
    } else {
        pterm.Warning.Println(color.YellowString("No additional notes provided for this release."))
    }

    return nil
}