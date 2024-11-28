package docker

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/containerregistry/armcontainerregistry"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ecr"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/api/types/registry"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/archive"
	"github.com/docker/docker/pkg/jsonmessage"
	"github.com/fatih/color"
	"github.com/pterm/pterm"
	"golang.org/x/oauth2/google"
)

// Build executes a Docker image build with comprehensive error handling and optimization
func Build(imageName, tag string, opts BuildOptions) error {
	runtime.GOMAXPROCS(runtime.NumCPU())

	ctx, cancel := context.WithTimeout(context.Background(), 25*time.Minute)
	defer cancel()

	cli, err := client.NewClientWithOpts(
		client.FromEnv,
		client.WithAPIVersionNegotiation(),
		client.WithTimeout(25*time.Minute),
	)
	if err != nil {
		return fmt.Errorf("docker client creation failed: %w", err)
	}
	defer cli.Close()

	var tarStream io.Reader
	var contextErr error
	var wg sync.WaitGroup
	errChan := make(chan error, 2)

	wg.Add(1)
	go func() {
		defer wg.Done()
		tarStream, contextErr = createOptimizedTarArchive(opts.ContextDir)
		if contextErr != nil {
			errChan <- fmt.Errorf("tar archive error: %w", contextErr)
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := validateBuildContext(opts.ContextDir); err != nil {
			errChan <- err
		}
	}()

	wg.Wait()
	close(errChan)

	for err := range errChan {
		if err != nil {
			return err
		}
	}

	buildOptions := types.ImageBuildOptions{
		Tags:        []string{fmt.Sprintf("%s:%s", imageName, tag)},
		Dockerfile:  filepath.Base(opts.DockerfilePath),
		NoCache:     opts.NoCache,
		BuildArgs:   convertToInterfaceMap(opts.BuildArgs),
		Target:      opts.Target,
		Remove:      true,
		ForceRemove: true,
		PullParent:  true,
		Platform:    opts.Platform,
	}

	spinner, _ := pterm.DefaultSpinner.Start("Docker build...")

	buildResponse, err := cli.ImageBuild(ctx, tarStream, buildOptions)
	if err != nil {
		spinner.Fail("Build initialization failed")
		return fmt.Errorf("image build error: %w", err)
	}
	defer buildResponse.Body.Close()

	err = jsonmessage.DisplayJSONMessagesStream(
		buildResponse.Body,
		os.Stdout,
		os.Stderr.Fd(),
		true,
		nil,
	)
	if err != nil {
		spinner.Fail("Build process encountered errors")
		return fmt.Errorf("build streaming error: %w", err)
	}

	spinner.Success("Docker image built successfully")
	color.Green("Successfully built %s:%s", imageName, tag)

	return nil
}


// Optimized tar archive creation
func createOptimizedTarArchive(contextDir string) (io.Reader, error) {
	return archive.Tar(contextDir, archive.Uncompressed)
}

// Context validation
func validateBuildContext(contextDir string) error {
	info, err := os.Stat(contextDir)
	if err != nil {
		return fmt.Errorf("invalid context directory: %w", err)
	}
	if !info.IsDir() {
		return fmt.Errorf("context must be a directory")
	}
	return nil
}

// Concurrent map conversion
func convertToInterfaceMap(args map[string]string) map[string]*string {
	result := make(map[string]*string, len(args))
	var mu sync.Mutex
	var wg sync.WaitGroup

	for key, value := range args {
		wg.Add(1)
		go func(k, v string) {
			defer wg.Done()
			mu.Lock()
			defer mu.Unlock()
			result[k] = &v
		}(key, value)
	}

	wg.Wait()
	return result
}

// BuildOptions struct to hold options for Docker build
type BuildOptions struct {
	ContextDir     string
	DockerfilePath string
	NoCache        bool
	BuildArgs      map[string]string
	Target         string
	Platform       string
}

// TagOptions struct to hold options for tagging a Docker image
type TagOptions struct {
	Source string
	Target string
}

// TagImage tags a local Docker image for use in a remote repository.
func TagImage(opts TagOptions) error {
	ctx := context.Background()
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		color.New(color.FgRed).Printf("Error creating Docker client: %v\n", err)
		return err
	}

	spinner, _ := pterm.DefaultSpinner.Start(fmt.Sprintf("Tagging image %s as %s...", opts.Source, opts.Target))
	if err := cli.ImageTag(ctx, opts.Source, opts.Target); err != nil {
		spinner.Fail(fmt.Sprintf("Failed to tag image: %v", err))
		color.New(color.FgRed).Println(err)
		return err
	}

	spinner.Success(fmt.Sprintf("Successfully tagged %s as %s", opts.Source, opts.Target))
	color.New(color.FgGreen).Printf("Image tagged successfully: %s -> %s\n", opts.Source, opts.Target)
	return nil
}

// PushOptions struct to hold options for pushing a Docker image
type PushOptions struct {
	ImageName string
}

// PushImage pushes a Docker image to a Docker registry.
func PushImage(opts PushOptions) error {
	ctx := context.Background()
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		pterm.Error.Println("Error creating Docker client:", err)
		return err
	}

	authConfig := registry.AuthConfig{
		Username: os.Getenv("DOCKER_USERNAME"),
		Password: os.Getenv("DOCKER_PASSWORD"),
	}
	encodedJSON, err := json.Marshal(authConfig)
	if err != nil {
		pterm.Error.Println("Error encoding auth config:", err)
		return err
	}
	authStr := base64.URLEncoding.EncodeToString(encodedJSON)

	spinner, _ := pterm.DefaultSpinner.Start(fmt.Sprintf("Pushing image %s...", opts.ImageName))
	options := image.PushOptions{
		RegistryAuth: authStr,
	}

	responseBody, err := cli.ImagePush(ctx, opts.ImageName, options)
	if err != nil {
		spinner.Fail("Failed to push the image: " + err.Error())
		return err
	}
	defer responseBody.Close()

	return handleDockerResponse(responseBody, spinner, opts)
}

func handleDockerResponse(responseBody io.ReadCloser, spinner *pterm.SpinnerPrinter, opts PushOptions) error {
	decoder := json.NewDecoder(responseBody)
	var lastProgress int
	for {
		var msg jsonmessage.JSONMessage
		if err := decoder.Decode(&msg); err == io.EOF {
			break
		} else if err != nil {
			pterm.Error.Println("Error decoding JSON:", err)
			return err
		}

		if msg.Error != nil {
			pterm.Error.Println("Error from Docker:", msg.Error.Message)
			return fmt.Errorf(msg.Error.Message)
		}

		if msg.Progress != nil && msg.Progress.Total > 0 {
			current := int(msg.Progress.Current * 100 / msg.Progress.Total)
			if current > lastProgress {
				progressMessage := fmt.Sprintf("Pushing image %s... %d%%", opts.ImageName, current)
				spinner.UpdateText(progressMessage)
				fmt.Printf("\r%s", pterm.Green(progressMessage))
				lastProgress = current
			}
		}

		if msg.Stream != "" {
			fmt.Print(pterm.Blue(msg.Stream))
		}
	}

	spinner.Success("Image push complete.")
	link := fmt.Sprint("https://hub.docker.com/repository/")
	pterm.Info.Println("Image Pushed on Docker Hub:", link)
	pterm.Success.Println("Successfully pushed image:", opts.ImageName)
	return nil
}

// Scout scans a Docker image for known vulnerabilities using 'docker scout cves'.
func Scout(dockerTag, sarifFile string) error {
	ctx := context.Background()

	args := []string{"scout", "cves", dockerTag}

	if sarifFile != "" {
		args = append(args, "--output", sarifFile)
	}

	cmd := exec.CommandContext(ctx, "docker", args...)

	var stdoutBuf, stderrBuf bytes.Buffer
	cmd.Stdout = &stdoutBuf
	cmd.Stderr = &stderrBuf

	spinner, _ := pterm.DefaultSpinner.Start("Running 'docker scout cves'")
	defer spinner.Stop()

	err := cmd.Run()

	spinner.Stop()

	outStr := stdoutBuf.String()
	errStr := stderrBuf.String()

	if err != nil {
		pterm.Error.Println("Error running 'docker scout cves':", err)
		if errStr != "" {
			pterm.Error.Println(errStr)
		}
		return fmt.Errorf("failed to run 'docker scout cves': %w", err)
	}

	if outStr != "" {
		pterm.Info.Println("Docker Scout CVEs output:")
		fmt.Println(color.YellowString(outStr))
	}

	if sarifFile != "" {
		if _, err := os.Stat(sarifFile); err == nil {
			pterm.Success.Println("SARIF report saved to:", sarifFile)
		} else {
			pterm.Warning.Println("Expected SARIF report not found at:", sarifFile)
		}
	}

	pterm.Success.Println("Scan completed successfully.")
	return nil
}

// RemoveImage removes a Docker image based on the provided flags.
func RemoveImage(imageTag string) error {
	ctx := context.Background()
	cli, err := client.NewClientWithOpts(client.WithAPIVersionNegotiation())
	if err != nil {
		return fmt.Errorf("failed to create Docker client: %w", err)
	}

	pterm.Info.Println("Removing local Docker image:", imageTag)
	spinner, _ := pterm.DefaultSpinner.Start("Removing image...")

	_, err = cli.ImageRemove(ctx, imageTag, image.RemoveOptions{Force: true})
	if err != nil {
		spinner.Fail("Failed to remove local Docker image:", imageTag)
		return fmt.Errorf("failed to remove local Docker image: %w", err)
	}

	spinner.Success("Successfully removed local Docker image:", imageTag)
	return nil
}

func PushImageToECR(imageName, region, repositoryName string) error {
	sess, err := session.NewSession(&aws.Config{
		Region: aws.String(region),
	})
	if err != nil {
		pterm.Error.Println(fmt.Errorf("failed to create AWS session: %w", err))
		return err
	}

	ecrClient := ecr.New(sess)

	describeRepositoriesInput := &ecr.DescribeRepositoriesInput{
		RepositoryNames: []*string{
			aws.String(repositoryName),
		},
	}
	_, err = ecrClient.DescribeRepositories(describeRepositoriesInput)
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok && aerr.Code() == ecr.ErrCodeRepositoryNotFoundException {
			createRepositoryInput := &ecr.CreateRepositoryInput{
				RepositoryName: aws.String(repositoryName),
			}
			_, err = ecrClient.CreateRepository(createRepositoryInput)
			if err != nil {
				pterm.Error.Println(fmt.Errorf("failed to create ECR repository: %w", err))
				return err
			}
			pterm.Info.Println("Created ECR repository:", repositoryName)
		} else {
			pterm.Error.Println(fmt.Errorf("failed to describe ECR repositories: %w", err))
			return err
		}
	}

	authTokenOutput, err := ecrClient.GetAuthorizationToken(&ecr.GetAuthorizationTokenInput{})
	if err != nil {
		pterm.Error.Println(fmt.Errorf("failed to get ECR authorization token: %w", err))
		return err
	}

	if len(authTokenOutput.AuthorizationData) == 0 {
		pterm.Error.Println("No authorization data received from ECR")
		return fmt.Errorf("no authorization data received from ECR")
	}

	authData := authTokenOutput.AuthorizationData[0]
	authToken, err := base64.StdEncoding.DecodeString(*authData.AuthorizationToken)
	if err != nil {
		pterm.Error.Println(fmt.Errorf("failed to decode authorization token: %w", err))
		return err
	}

	credentials := strings.SplitN(string(authToken), ":", 2)
	if len(credentials) != 2 {
		pterm.Error.Println("Invalid authorization token format")
		return fmt.Errorf("invalid authorization token format")
	}

	ecrURL := strings.TrimPrefix(*authData.ProxyEndpoint, "https://")

	pterm.Info.Println("Initializing Docker client...")
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		pterm.Error.Println(fmt.Errorf("failed to create Docker client: %w", err))
		return err
	}

	authConfig := registry.AuthConfig{
		Username:      credentials[0],
		Password:      credentials[1],
		ServerAddress: *authData.ProxyEndpoint,
	}

	pterm.Info.Println("Authenticating Docker client to ECR...")

	encodedJSON, err := json.Marshal(authConfig)
	if err != nil {
		pterm.Error.Println(fmt.Errorf("failed to encode auth config: %w", err))
		return err
	}
	authStr := base64.URLEncoding.EncodeToString(encodedJSON)

	ecrImage := fmt.Sprintf("%s/%s", ecrURL, repositoryName)
	pterm.Info.Println("Tagging image for ECR...")
	if err := cli.ImageTag(context.Background(), imageName, ecrImage); err != nil {
		pterm.Error.Println(fmt.Errorf("failed to tag image: %w", err))
		return err
	}

	pterm.Info.Println("Pushing image to ECR...")
	pushResponse, err := cli.ImagePush(context.Background(), ecrImage, image.PushOptions{
		RegistryAuth: authStr,
	})
	if err != nil {
		pterm.Error.Println(fmt.Errorf("failed to push image to ECR: %w", err))
		return err
	}
	defer pushResponse.Close()

	decoder := json.NewDecoder(pushResponse)
	for {
		var message map[string]interface{}
		if err := decoder.Decode(&message); err != nil {
			if err == io.EOF {
				break
			}
			pterm.Error.Println(fmt.Errorf("error decoding JSON message from push: %w", err))
			return err
		}

		if status, ok := message["status"].(string); ok {
			if status != "Waiting" {
				pterm.Info.Println(status)
			}
		}
		if errorDetail, ok := message["errorDetail"].(map[string]interface{}); ok {
			pterm.Error.Println(fmt.Errorf("error pushing image: %v", errorDetail["message"]))
			return fmt.Errorf("error pushing image: %v", errorDetail["message"])
		}
	}

	link := fmt.Sprintf("https://%s.console.aws.amazon.com/ecr/repositories/%s", region, repositoryName)
	pterm.Info.Println("Image pushed to ECR:", link)
	fmt.Println()
	pterm.Success.Println("Image successfully pushed to ECR:", ecrImage)
	return nil
}

func PushImageToACR(subscriptionID, resourceGroupName, registryName, imageName string) error {
	ctx := context.Background()

	spinner, _ := pterm.DefaultSpinner.Start("Authenticating with Azure...")
	cred, err := azidentity.NewDefaultAzureCredential(nil)
	if err != nil {
		spinner.Fail("Failed to authenticate with Azure")
		color.New(color.FgRed).Printf("Error: %v\n", err)
		return err
	}
	spinner.Success("Authenticated with Azure")

	spinner, _ = pterm.DefaultSpinner.Start("Creating registry client...")
	registryClient, err := armcontainerregistry.NewRegistriesClient(subscriptionID, cred, nil)
	if err != nil {
		spinner.Fail("Failed to create registry client")
		color.New(color.FgRed).Printf("Error: %v\n", err)
		return err
	}
	spinner.Success("Registry client created")

	spinner, _ = pterm.DefaultSpinner.Start("Retrieving registry details...")
	registryResp, err := registryClient.Get(ctx, resourceGroupName, registryName, nil)
	if err != nil {
		spinner.Fail("Failed to retrieve registry details")
		color.New(color.FgRed).Printf("Error: %v\n", err)
		return err
	}
	loginServer := *registryResp.Properties.LoginServer
	spinner.Success("Registry details retrieved")

	spinner, _ = pterm.DefaultSpinner.Start("Retrieving registry credentials...")
	credentialsResp, err := registryClient.ListCredentials(ctx, resourceGroupName, registryName, nil)
	if err != nil {
		spinner.Fail("Failed to retrieve registry credentials")
		color.New(color.FgRed).Printf("Error: %v\n", err)
		return err
	}
	if credentialsResp.Username == nil || len(credentialsResp.Passwords) == 0 || credentialsResp.Passwords[0].Value == nil {
		spinner.Fail("Registry credentials are not available")
		color.New(color.FgRed).Println("Error: Registry credentials are not available")
		return fmt.Errorf("registry credentials are not available")
	}
	username := *credentialsResp.Username
	password := *credentialsResp.Passwords[0].Value
	spinner.Success("Registry credentials retrieved")

	spinner, _ = pterm.DefaultSpinner.Start("Creating Docker client...")
	dockerClient, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		spinner.Fail("Failed to create Docker client")
		color.New(color.FgRed).Printf("Error: %v\n", err)
		return err
	}
	spinner.Success("Docker client created")

	spinner, _ = pterm.DefaultSpinner.Start("Tagging the image...")
	taggedImage := fmt.Sprintf("%s/%s", loginServer, imageName)
	err = dockerClient.ImageTag(ctx, imageName, taggedImage)
	if err != nil {
		spinner.Fail("Failed to tag the image")
		color.New(color.FgRed).Printf("Error: %v\n", err)
		return err
	}
	spinner.Success("Image tagged")

	spinner, _ = pterm.DefaultSpinner.Start("Pushing the image to ACR...")
	authConfig := registry.AuthConfig{
		Username:      username,
		Password:      password,
		ServerAddress: loginServer,
	}
	encodedAuth, err := encodeAuthToBase64(authConfig)
	if err != nil {
		spinner.Fail("Failed to encode authentication credentials")
		color.New(color.FgRed).Printf("Error: %v\n", err)
		return err
	}

	pushOptions := image.PushOptions{
		RegistryAuth: encodedAuth,
	}

	pushResponse, err := dockerClient.ImagePush(ctx, taggedImage, pushOptions)
	if err != nil {
		spinner.Fail("Failed to push the image")
		color.New(color.FgRed).Printf("Error: %v\n", err)
		return err
	}
	defer pushResponse.Close()

	dec := json.NewDecoder(pushResponse)
	for {
		var event jsonmessage.JSONMessage
		if err := dec.Decode(&event); err != nil {
			if err == io.EOF {
				break
			}
			spinner.Fail("Failed to read push response")
			color.New(color.FgRed).Printf("Error: %v\n", err)
			return err
		}
		if event.Error != nil {
			spinner.Fail("Failed to push the image")
			color.New(color.FgRed).Printf("Error: %v\n", event.Error)
			return event.Error
		}
		if event.Status != "" {
			spinner.UpdateText(event.Status)
		}
	}
	spinner.Success("Image pushed to ACR")
	link := fmt.Sprintf("https://%s.azurecr.io", registryName)
	color.New(color.FgGreen).Printf("Image pushed to ACR: %s\n", link)
	color.New(color.FgGreen).Printf("Successfully pushed image '%s' to ACR '%s'\n", imageName, registryName)
	return nil
}

func PushImageToGCR(projectID, imageName string) error {
	ctx := context.Background()

	spinner, _ := pterm.DefaultSpinner.Start("Authenticating with Google Cloud...")
	creds, err := google.FindDefaultCredentials(ctx, "https://www.googleapis.com/auth/cloud-platform")
	if err != nil {
		spinner.Fail("Failed to authenticate with Google Cloud")
		color.New(color.FgRed).Printf("Error: %v\n", err)
		return err
	}
	spinner.Success("Authenticated with Google Cloud")

	spinner, _ = pterm.DefaultSpinner.Start("Obtaining access token...")
	tokenSource := creds.TokenSource
	token, err := tokenSource.Token()
	if err != nil {
		spinner.Fail("Failed to obtain access token")
		color.New(color.FgRed).Printf("Error: %v\n", err)
		return err
	}
	spinner.Success("Access token obtained")

	spinner, _ = pterm.DefaultSpinner.Start("Creating Docker client...")
	dockerClient, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		spinner.Fail("Failed to create Docker client")
		color.New(color.FgRed).Printf("Error: %v\n", err)
		return err
	}
	spinner.Success("Docker client created")

	spinner, _ = pterm.DefaultSpinner.Start("Tagging the image...")
	var registryHost string
	if strings.Contains(imageName, "gcr.io") {
		registryHost = "gcr.io"
	} else {
		registryHost = fmt.Sprintf("gcr.io/%s", projectID)
	}
	taggedImage := fmt.Sprintf("%s/%s", registryHost, imageName)
	err = dockerClient.ImageTag(ctx, imageName, taggedImage)
	if err != nil {
		spinner.Fail("Failed to tag the image")
		color.New(color.FgRed).Printf("Error: %v\n", err)
		return err
	}
	spinner.Success("Image tagged")

	spinner, _ = pterm.DefaultSpinner.Start("Pushing the image to GCR...")
	authConfig := registry.AuthConfig{
		Username:      "oauth2accesstoken",
		Password:      token.AccessToken,
		ServerAddress: "https://gcr.io",
	}
	encodedAuth, err := encodeAuthToBase64(authConfig)
	if err != nil {
		spinner.Fail("Failed to encode authentication credentials")
		color.New(color.FgRed).Printf("Error: %v\n", err)
		return err
	}

	pushOptions := image.PushOptions{
		RegistryAuth: encodedAuth,
	}

	pushResponse, err := dockerClient.ImagePush(ctx, taggedImage, pushOptions)
	if err != nil {
		spinner.Fail("Failed to push the image")
		color.New(color.FgRed).Printf("Error: %v\n", err)
		return err
	}
	defer pushResponse.Close()

	dec := json.NewDecoder(pushResponse)
	progressBar, _ := pterm.DefaultProgressbar.WithTotal(100).WithTitle("Pushing to GCR").Start()
	for {
		var event jsonmessage.JSONMessage
		if err := dec.Decode(&event); err != nil {
			if err == io.EOF {
				break
			}
			spinner.Fail("Failed to read push response")
			color.New(color.FgRed).Printf("Error: %v\n", err)
			return err
		}
		if event.Error != nil {
			spinner.Fail("Failed to push the image")
			color.New(color.FgRed).Printf("Error: %v\n", event.Error)
			return event.Error
		}
		if event.Progress != nil && event.Progress.Total > 0 {
			progress := int(float64(event.Progress.Current) / float64(event.Progress.Total) * 100)
			if progress > 100 {
				progress = 100
			}
			progressBar.Add(progress - progressBar.Current)
		}
	}
	progressBar.Stop()
	spinner.Success("Image pushed to GCR")

	link := fmt.Sprintf("https://console.cloud.google.com/gcr/images/%s/%s?project=%s", projectID, imageName, projectID)
	color.New(color.FgGreen).Printf("Image pushed to GCR: %s\n", link)
	color.New(color.FgGreen).Printf("Successfully pushed image '%s' to GCR\n", taggedImage)
	return nil
}

func encodeAuthToBase64(authConfig registry.AuthConfig) (string, error) {
	authJSON, err := json.Marshal(authConfig)
	if err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(authJSON), nil
}
