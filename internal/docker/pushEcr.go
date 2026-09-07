package docker

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ecr"
	"github.com/clouddrove/smurf/configs"
	"github.com/clouddrove/smurf/internal/ai"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/api/types/registry"
	"github.com/docker/docker/client"
)

type ECRLogger struct {
	startTime time.Time
}

func NewECRLogger() *ECRLogger {
	return &ECRLogger{startTime: time.Now()}
}

func (l *ECRLogger) logStep(message string) {
	fmt.Printf("%s[%s] %s %s%s%s\n",
		colorBlue,
		time.Since(l.startTime).Round(time.Millisecond),
		"→",
		colorCyan,
		message,
		colorReset)
}

func (l *ECRLogger) logSuccess(message string) {
	fmt.Printf("%s[%s] %s %s%s%s\n",
		colorGreen,
		time.Since(l.startTime).Round(time.Millisecond),
		"✓",
		colorGreen,
		message,
		colorReset)
}

func (l *ECRLogger) logLayerPushed(layerID string) {
	shortID := layerID
	if len(shortID) > 12 {
		shortID = shortID[:12]
	}
	fmt.Printf("%s[%s] %s %s%s pushed%s\n",
		colorYellow,
		time.Since(l.startTime).Round(time.Millisecond),
		"⬆",
		colorCyan,
		shortID+"...",
		colorReset)
}

func (l *ECRLogger) logError(message string, err error) {
	fmt.Printf("%s[%s] %s %s%s: %v%s\n",
		colorRed,
		time.Since(l.startTime).Round(time.Millisecond),
		"✗",
		colorRed,
		message,
		err,
		colorReset)
}

func PushImageToECR(imageName, region, repositoryName string, useAI bool) error {
	logger := NewECRLogger()
	ctx := context.Background()

	logger.logStep("Starting image push to ECR")

	// AWS Session
	sess, err := session.NewSession(&aws.Config{
		Region: aws.String(region),
	})
	if err != nil {
		logger.logError("Failed to create AWS session", err)
		ai.AIExplainError(useAI, err.Error())
		return fmt.Errorf("failed to create AWS session: %w", err)
	}

	ecrClient := ecr.New(sess)

	// Check if repository exists
	describeRepositoriesInput := &ecr.DescribeRepositoriesInput{
		RepositoryNames: []*string{aws.String(repositoryName)},
	}
	_, err = ecrClient.DescribeRepositories(describeRepositoriesInput)
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok && aerr.Code() == ecr.ErrCodeRepositoryNotFoundException {
			// Create repository if it doesn't exist
			createRepositoryInput := &ecr.CreateRepositoryInput{
				RepositoryName: aws.String(repositoryName),
			}
			_, err = ecrClient.CreateRepository(createRepositoryInput)
			if err != nil {
				logger.logError("Failed to create ECR repository", err)
				ai.AIExplainError(useAI, err.Error())
				return fmt.Errorf("failed to create ECR repository: %w", err)
			}
			logger.logSuccess(fmt.Sprintf("Created ECR repository: %s%s%s", colorCyan, repositoryName, colorReset))
		} else {
			logger.logError("Failed to describe ECR repositories", err)
			ai.AIExplainError(useAI, err.Error())
			return fmt.Errorf("failed to describe ECR repositories: %w", err)
		}
	}

	// Get ECR auth token
	authTokenOutput, err := ecrClient.GetAuthorizationToken(&ecr.GetAuthorizationTokenInput{})
	if err != nil {
		logger.logError("Failed to get ECR authorization token", err)
		ai.AIExplainError(useAI, err.Error())
		return fmt.Errorf("failed to get ECR authorization token: %w", err)
	}
	if len(authTokenOutput.AuthorizationData) == 0 {
		logger.logError("No authorization data received from ECR", nil)
		return fmt.Errorf("no authorization data received from ECR")
	}

	authData := authTokenOutput.AuthorizationData[0]
	authToken, err := base64.StdEncoding.DecodeString(*authData.AuthorizationToken)
	if err != nil {
		logger.logError("Failed to decode authorization token", err)
		ai.AIExplainError(useAI, err.Error())
		return fmt.Errorf("failed to decode authorization token: %w", err)
	}

	username, password, err := splitECRCredentials(string(authToken))
	if err != nil {
		logger.logError("Invalid authorization token format", nil)
		return err
	}

	ecrURL := ecrRegistryHost(*authData.ProxyEndpoint)

	// Docker client
	cli, err := client.NewClientWithOpts(clientOpts()...)
	if err != nil {
		logger.logError("Failed to create Docker client", err)
		ai.AIExplainError(useAI, err.Error())
		return fmt.Errorf("failed to create Docker client: %w", err)
	}

	// Docker auth
	authConfig := registry.AuthConfig{
		Username:      username,
		Password:      password,
		ServerAddress: *authData.ProxyEndpoint,
	}
	encodedJSON, err := json.Marshal(authConfig)
	if err != nil {
		logger.logError("Failed to encode auth config", err)
		ai.AIExplainError(useAI, err.Error())
		return fmt.Errorf("failed to encode auth config: %w", err)
	}
	authStr := base64.URLEncoding.EncodeToString(encodedJSON)

	// Tag image
	_, tag, _ := configs.ParseImage(imageName)
	ecrImage := ecrImageRef(ecrURL, repositoryName, tag)
	if err := cli.ImageTag(ctx, imageName, ecrImage); err != nil {
		logger.logError("Failed to tag image", err)
		ai.AIExplainError(useAI, err.Error())
		return fmt.Errorf("failed to tag image: %w", err)
	}
	logger.logSuccess(fmt.Sprintf("Tagged image: %s%s%s", colorCyan, ecrImage, colorReset))

	// Push image
	pushResponse, err := cli.ImagePush(ctx, ecrImage, image.PushOptions{
		RegistryAuth: authStr,
	})
	if err != nil {
		logger.logError("Failed to push image to ECR", err)
		ai.AIExplainError(useAI, err.Error())
		return fmt.Errorf("failed to push image to ECR: %w", err)
	}
	defer pushResponse.Close()

	logger.logStep("Starting image push")
	decoder := json.NewDecoder(pushResponse)
	var lastError error

	for {
		var message map[string]interface{}
		if err := decoder.Decode(&message); err != nil {
			if err == io.EOF {
				break
			}
			logger.logError("Error decoding JSON message from push", err)
			ai.AIExplainError(useAI, err.Error())
			return fmt.Errorf("error decoding JSON message from push: %w", err)
		}

		if errorDetail, ok := message["errorDetail"].(map[string]interface{}); ok {
			lastError = fmt.Errorf("error pushing image: %v", errorDetail["message"])
			logger.logError("Push failed", lastError)
			return lastError
		}

		// Only log when a layer is fully pushed
		if status, ok := message["status"].(string); ok && status == "Pushed" {
			if id, ok := message["id"].(string); ok {
				logger.logLayerPushed(id)
			}
		}
	}

	if lastError == nil {
		logger.logSuccess("Image successfully pushed to ECR")
		link := fmt.Sprintf("https://%s.console.aws.amazon.com/ecr/repositories/%s", region, repositoryName)
		logger.logSuccess(fmt.Sprintf("View in console: %s%s%s", colorCyan, link, colorReset))
		logger.logSuccess(fmt.Sprintf("Image reference: %s%s%s", colorCyan, ecrImage, colorReset))
	}

	return nil
}

// The three steps below decide which registry the image is pushed to and under
// what credentials. They were inlined in PushImageToECR, which cannot run
// without AWS, so the parts of this file that decide where an image ends up
// were the only ones with no test at all. Getting any of them wrong pushes to
// the wrong place or fails to authenticate, and both are quiet failures.

// splitECRCredentials splits the decoded ECR authorization token, which AWS
// returns as "user:password".
//
// SplitN with a limit of two matters: a password may itself contain a colon,
// and splitting on every colon would truncate it into an authentication
// failure that looks like a permissions problem.
func splitECRCredentials(token string) (username, password string, err error) {
	parts := strings.SplitN(token, ":", 2)
	if len(parts) != 2 {
		return "", "", fmt.Errorf("invalid authorization token format")
	}
	return parts[0], parts[1], nil
}

// ecrRegistryHost turns the proxy endpoint AWS returns into the host used in
// an image reference. The endpoint arrives as a URL; an image reference cannot
// carry the scheme.
func ecrRegistryHost(proxyEndpoint string) string {
	host := strings.TrimPrefix(proxyEndpoint, "https://")
	host = strings.TrimPrefix(host, "http://")
	return strings.TrimSuffix(host, "/")
}

// ecrImageRef builds the fully qualified reference an image is tagged with
// before being pushed.
func ecrImageRef(registryHost, repositoryName, tag string) string {
	if tag == "" {
		tag = "latest"
	}
	return fmt.Sprintf("%s/%s:%s", registryHost, repositoryName, tag)
}
