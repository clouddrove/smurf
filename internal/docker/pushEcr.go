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
		pterm.Error.Println("failed to create AWS session : ", err)
		return fmt.Errorf("failed to create AWS session : %v", err)
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
				pterm.Error.Println("failed to create ECR repository : ", err)
				return fmt.Errorf("failed to create ECR repository : %v", err)
			}
			pterm.Info.Println("Created ECR repository:", repositoryName)
		} else {
			pterm.Error.Println("failed to describe ECR repositories : ", err)
			return fmt.Errorf("failed to describe ECR repositories : %v", err)
		}
	}

	authTokenOutput, err := ecrClient.GetAuthorizationToken(&ecr.GetAuthorizationTokenInput{})
	if err != nil {
		pterm.Error.Println("failed to get ECR authorization token : ", err)
		return fmt.Errorf("failed to get ECR authorization token : %v", err)
	}
	if len(authTokenOutput.AuthorizationData) == 0 {
		pterm.Error.Println("No authorization data received from ECR")
		return fmt.Errorf("no authorization data received from ECR : %v", 0)
	}

	authData := authTokenOutput.AuthorizationData[0]
	authToken, err := base64.StdEncoding.DecodeString(*authData.AuthorizationToken)
	if err != nil {
		pterm.Error.Println("failed to decode authorization token: ", err)
		return fmt.Errorf("failed to decode authorization token: %v", err)
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
		pterm.Error.Println("failed to create Docker client: ", err)
		return fmt.Errorf("failed to create Docker client: %v", err)
	}

	authConfig := registry.AuthConfig{
		Username:      credentials[0],
		Password:      credentials[1],
		ServerAddress: *authData.ProxyEndpoint,
	}
	pterm.Info.Println("Authenticating Docker client to ECR...")
	encodedJSON, err := json.Marshal(authConfig)
	if err != nil {
		pterm.Error.Println("failed to encode auth config: ", err)
		return fmt.Errorf("failed to encode auth config: %v", err)
	}
	authStr := base64.URLEncoding.EncodeToString(encodedJSON)

	_, tag := parseImageNameAndTag(imageName)
	ecrImage := fmt.Sprintf("%s/%s:%s", ecrURL, repositoryName, tag)
	pterm.Info.Println("Tagging image for ECR...")
	if err := cli.ImageTag(context.Background(), imageName, ecrImage); err != nil {
		pterm.Error.Println("failed to tag image: ", err)
		return fmt.Errorf("failed to tag image: %v", err)
	}

	pterm.Info.Println("Starting push to ECR...")
	pushResponse, err := cli.ImagePush(context.Background(), ecrImage, image.PushOptions{
		RegistryAuth: authStr,
	})
	if err != nil {
		pterm.Error.Println("Failed to push image to ECR: ", err)
		return fmt.Errorf("failed to push image to ECR: %v", err)
	}
	defer pushResponse.Close()

	pterm.Info.Println("Pushing in progress...")
	decoder := json.NewDecoder(pushResponse)
	var lastError error

	for {
		var message map[string]interface{}
		if err := decoder.Decode(&message); err != nil {
			if err == io.EOF {
				break
			}
			pterm.Error.Println("Error decoding JSON message from push: ", err)
			return fmt.Errorf("error decoding JSON message from push: %v", err)
		}

		if errorDetail, ok := message["errorDetail"].(map[string]interface{}); ok {
			lastError = fmt.Errorf("error pushing image: %v", errorDetail["message"])
			pterm.Error.Println("failed to tag image: ", lastError)
			return fmt.Errorf("failed to tag image: %v", lastError)
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
				pterm.Info.Println(fmt.Sprintf("[%s] %s %s\n", id, status, progress))
			} else if id != "" {
				pterm.Info.Println(fmt.Sprintf("[%s] %s\n", id, status))
			} else {
				pterm.Info.Println(status)
			}
		}
	}

	if lastError == nil {
		pterm.Success.Println("Image successfully pushed to ECR:", ecrImage)
	}

	link := fmt.Sprintf("https://%s.console.aws.amazon.com/ecr/repositories/%s", region, repositoryName)
	pterm.Info.Println("Image pushed to ECR. View it here:", link)
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
