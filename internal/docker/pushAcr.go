package docker

import (
	"context"
	"encoding/json"
	"fmt"
	"io"

	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/containerregistry/armcontainerregistry"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/api/types/registry"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/jsonmessage"
	"github.com/pterm/pterm"
)

// PushImageToACR pushes the specified Docker image to the specified Azure Container Registry.
// It authenticates with Azure, retrieves the registry details and credentials, tags the image,
// and pushes it to the registry. It displays a spinner with progress updates and prints the
// push response messages. Upon successful completion, it prints a success message with a link
// to the pushed image in the ACR.
func PushImageToACR(subscriptionID, resourceGroupName, registryName, imageName string) error {
	ctx := context.Background()

	spinner, _ := pterm.DefaultSpinner.Start("Authenticating with Azure...")
	cred, err := azidentity.NewDefaultAzureCredential(nil)
	if err != nil {
		spinner.Fail("Failed to authenticate with Azure\n")
		return fmt.Errorf("failed to authenticate with Azure : %v", err)
	}
	spinner.Success("Authenticated with Azure\n")

	spinner, _ = pterm.DefaultSpinner.Start("Creating registry client...")
	registryClient, err := armcontainerregistry.NewRegistriesClient(subscriptionID, cred, nil)
	if err != nil {
		spinner.Fail("Failed to create registry client\n")
		return fmt.Errorf("failed to create registry client : %v", err)
	}
	spinner.Success("Registry client created\n")

	spinner, _ = pterm.DefaultSpinner.Start("Retrieving registry details...")
	registryResp, err := registryClient.Get(ctx, resourceGroupName, registryName, nil)
	if err != nil {
		spinner.Fail("Failed to retrieve registry details\n")
		return fmt.Errorf("failed to retrieve registry details : %v", err)
	}
	loginServer := *registryResp.Properties.LoginServer
	spinner.Success("Registry details retrieved\n")

	spinner, _ = pterm.DefaultSpinner.Start("Retrieving registry credentials...")
	credentialsResp, err := registryClient.ListCredentials(ctx, resourceGroupName, registryName, nil)
	if err != nil {
		spinner.Fail("Failed to retrieve registry credentials\n")
		return fmt.Errorf("failed to retrieve registry credentials : %v", err)
	}
	if credentialsResp.Username == nil || len(credentialsResp.Passwords) == 0 || credentialsResp.Passwords[0].Value == nil {
		spinner.Fail("Registry credentials are not available\n")
		return fmt.Errorf("registry credentials are not available")
	}
	username := *credentialsResp.Username
	password := *credentialsResp.Passwords[0].Value
	spinner.Success("Registry credentials retrieved\n")

	spinner, _ = pterm.DefaultSpinner.Start("Creating Docker client...")
	dockerClient, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		spinner.Fail("Failed to create Docker client\n")
		return fmt.Errorf("failed to create Docker client : %v", err)
	}
	spinner.Success("Docker client created\n")

	spinner, _ = pterm.DefaultSpinner.Start("Tagging the image...")
	taggedImage := fmt.Sprintf("%s/%s", loginServer, imageName)
	err = dockerClient.ImageTag(ctx, imageName, taggedImage)
	if err != nil {
		spinner.Fail("Failed to tag the image\n")
		return fmt.Errorf("failed to tag the image : %v", err)
	}
	spinner.Success("Image tagged\n")

	spinner, _ = pterm.DefaultSpinner.Start("Pushing the image to ACR...")
	authConfig := registry.AuthConfig{
		Username:      username,
		Password:      password,
		ServerAddress: loginServer,
	}
	encodedAuth, err := encodeAuthToBase64(authConfig)
	if err != nil {
		spinner.Fail("Failed to encode authentication credentials\n")
		return fmt.Errorf("failed to encode authentication credentials : %v", err)
	}

	pushOptions := image.PushOptions{
		RegistryAuth: encodedAuth,
	}

	pushResponse, err := dockerClient.ImagePush(ctx, taggedImage, pushOptions)
	if err != nil {
		spinner.Fail("Failed to push the image\n")
		return fmt.Errorf("failed to push the image : %v", err)
	}
	defer pushResponse.Close()

	dec := json.NewDecoder(pushResponse)
	for {
		var event jsonmessage.JSONMessage
		if err := dec.Decode(&event); err != nil {
			if err == io.EOF {
				break
			}
			spinner.Fail("Failed to read push response\n")
			return fmt.Errorf("failed to read push response : %v", err)
		}
		if event.Error != nil {
			spinner.Fail("Failed to push the image\n")
			return fmt.Errorf("failed to push the image : %v", event.Error)
		}
		if event.Status != "" {
			spinner.UpdateText(event.Status)
		}
	}
	spinner.Success("Image pushed to ACR\n")
	link := fmt.Sprintf("https://%s.azurecr.io", registryName)
	pterm.Success.Printfln("Image pushed to ACR: %s\n", link)
	pterm.Success.Printfln("Successfully pushed image '%s' to ACR '%s'\n", imageName, registryName)
	return nil
}
