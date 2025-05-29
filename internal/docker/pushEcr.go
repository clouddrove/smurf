package docker

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ecr"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/api/types/registry"
	"github.com/docker/docker/client"
	"github.com/fatih/color"
	"github.com/pterm/pterm"
)

// PushImageToECR pushes the specified Docker image to the specified AWS Elastic Container Registry.
// It authenticates with AWS, retrieves the registry details and credentials, tags the image,
// and pushes it to the registry. It displays a spinner with progress updates and prints the
// push response messages. Upon successful completion, it prints a success message with a link
// to the pushed image in the ECR.
func PushImageToECR(imageName, region, repositoryName string) error {
	sess, err := session.NewSession(&aws.Config{
		Region: aws.String(region),
	})
	if err != nil {
		pterm.Error.Println(color.RedString("failed to create AWS session: %w", err))
		return err
	}

	ecrClient := ecr.New(sess)

	describeRepositoriesInput := &ecr.DescribeRepositoriesInput{
		RepositoryNames: []*string{aws.String(repositoryName)},
	}
	_, err = ecrClient.DescribeRepositories(describeRepositoriesInput)
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok && aerr.Code() == ecr.ErrCodeRepositoryNotFoundException {
			createRepositoryInput := &ecr.CreateRepositoryInput{
				RepositoryName: aws.String(repositoryName),
			}
			_, err = ecrClient.CreateRepository(createRepositoryInput)
			if err != nil {
				pterm.Error.Println(color.RedString("failed to create ECR repository: %w", err))
				return err
			}
			pterm.Info.Println("Created ECR repository:", repositoryName)
		} else {
			pterm.Error.Println(color.RedString("failed to describe ECR repositories: %w", err))
			return err
		}
	}

	authTokenOutput, err := ecrClient.GetAuthorizationToken(&ecr.GetAuthorizationTokenInput{})
	if err != nil {
		pterm.Error.Println(color.RedString("failed to get ECR authorization token: %w", err))
		return err
	}
	if len(authTokenOutput.AuthorizationData) == 0 {
		pterm.Error.Println(color.RedString("No authorization data received from ECR"))
		return fmt.Errorf("no authorization data received from ECR")
	}

	authData := authTokenOutput.AuthorizationData[0]
	authToken, err := base64.StdEncoding.DecodeString(*authData.AuthorizationToken)
	if err != nil {
		pterm.Error.Println(color.RedString("failed to decode authorization token: %w", err))
		return err
	}

	credentials := strings.SplitN(string(authToken), ":", 2)
	if len(credentials) != 2 {
		pterm.Error.Println(color.RedString("Invalid authorization token format"))
		return fmt.Errorf("invalid authorization token format")
	}

	ecrURL := strings.TrimPrefix(*authData.ProxyEndpoint, "https://")

	pterm.Info.Println(color.CyanString("Initializing Docker client..."))
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
	pterm.Info.Println(color.CyanString("Authenticating Docker client to ECR..."))
	encodedJSON, err := json.Marshal(authConfig)
	if err != nil {
		pterm.Error.Println(color.RedString("failed to encode auth config: %w", err))
		return err
	}
	authStr := base64.URLEncoding.EncodeToString(encodedJSON)

	_, tag := parseImageNameAndTag(imageName)
	ecrImage := fmt.Sprintf("%s/%s:%s", ecrURL, repositoryName, tag)
	pterm.Info.Println(color.CyanString("Tagging image for ECR..."))
	if err := cli.ImageTag(context.Background(), imageName, ecrImage); err != nil {
		pterm.Error.Println(color.RedString("failed to tag image: %w", err))
		return err
	}

	pterm.Info.Println(color.CyanString("Starting push to ECR..."))
	pushResponse, err := cli.ImagePush(context.Background(), ecrImage, image.PushOptions{
		RegistryAuth: authStr,
	})
	if err != nil {
		pterm.Error.Println(color.RedString("Failed to push image to ECR:", err))
		return err
	}
	defer pushResponse.Close()

	pterm.Info.Println(color.CyanString("Pushing in progress..."))
	decoder := json.NewDecoder(pushResponse)
	var lastError error

	for {
		var message map[string]interface{}
		if err := decoder.Decode(&message); err != nil {
			if err == io.EOF {
				break
			}
			pterm.Error.Println(color.RedString("Error decoding JSON message from push:", err))
			return err
		}

		if errorDetail, ok := message["errorDetail"].(map[string]interface{}); ok {
			lastError = fmt.Errorf("error pushing image: %v", errorDetail["message"])
			pterm.Error.Println(color.RedString("%v", lastError))
			return lastError
		}

		if status, ok := message["status"].(string); ok {
			id := ""
			if val, found := message["id"].(string); found {
				id = val
			}
			progress := ""
			if val, found := message["progress"].(string); found {
				progress = val
			}

			if id != "" && progress != "" {
				pterm.Info.Println(color.CyanString("[%s] %s %s\n", id, status, progress))
			} else if id != "" {
				pterm.Info.Println(color.CyanString("[%s] %s\n", id, status))
			} else {
				pterm.Info.Println(color.CyanString(status))
			}
		}
	}

	if lastError == nil {
		pterm.Success.Println("Image successfully pushed to ECR:", ecrImage)
	}

	link := fmt.Sprintf("https://%s.console.aws.amazon.com/ecr/repositories/%s", region, repositoryName)
	pterm.Info.Println(color.CyanString("Image pushed to ECR. View it here:", link))
	return nil
}

// parseImageNameAndTag splits an image name into the repository part and the tag part.
// If no tag is found, it defaults to "latest".
func parseImageNameAndTag(imageName string) (string, string) {
	parts := strings.Split(imageName, ":")
	if len(parts) == 2 {
		return parts[0], parts[1]
	}
	return imageName, "latest"
}
