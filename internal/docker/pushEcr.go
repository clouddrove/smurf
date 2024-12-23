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
	"github.com/aws/aws-sdk-go/service/ecr"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/api/types/registry"
	"github.com/docker/docker/client"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/pterm/pterm"
)

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
