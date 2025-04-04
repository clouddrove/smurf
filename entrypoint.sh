#!/bin/bash
set -e

# Debugging info
echo "Starting entrypoint script..."
echo "Current User: $(whoami)"
echo "Working Directory: $(pwd)"
echo "AWS Region: ${AWS_REGION:-not set}"
echo "EKS Cluster: ${EKS_CLUSTER_NAME:-not set}"

# Function to check file existence and non-emptiness
check_file_exists() {
    if [ ! -s "$1" ]; then
        echo "Error: $1 is either missing or empty."
        exit 1
    fi
}

aws_eks_login() {
    if [[ -z "$AWS_ACCESS_KEY_ID" || -z "$AWS_SECRET_ACCESS_KEY" || -z "$AWS_DEFAULT_REGION" || -z "$EKS_CLUSTER_NAME" ]]; then
        echo "⚠️ Warning: Required environment variables not set. Please ensure the following are set:"
        echo "  - AWS_ACCESS_KEY_ID"
        echo "  - AWS_SECRET_ACCESS_KEY"
        echo "  - AWS_DEFAULT_REGION"
        echo "  - EKS_CLUSTER_NAME"
        echo "Skipping AWS and EKS login."
        return 1
    fi

    echo "🔹 Configuring AWS credentials..."
    aws configure set aws_access_key_id "$AWS_ACCESS_KEY_ID"
    aws configure set aws_secret_access_key "$AWS_SECRET_ACCESS_KEY"
    aws configure set default.region "$AWS_DEFAULT_REGION"
    echo "✅ AWS credentials configured successfully."

    # EKS Cluster Login
    echo "🔹 Getting EKS token for cluster: $EKS_CLUSTER_NAME..."
    aws eks update-kubeconfig --region "$AWS_DEFAULT_REGION" --name "$EKS_CLUSTER_NAME"
    if [ $? -eq 0 ]; then
        echo "✅ Successfully configured EKS cluster access."
    else
        echo "❌ Failed to configure EKS cluster access."
        return 1
    fi
}

gcp_gke_login() {
    if [[ -z "$GCP_SERVICE_ACCOUNT_KEY" || -z "$GCP_PROJECT_ID" || -z "$GCP_REGION" || -z "$GKE_CLUSTER_NAME" ]]; then
        echo "⚠️ Warning: Required environment variables not set. Please ensure the following are set:"
        echo "  - GCP_SERVICE_ACCOUNT_KEY (path to service account key file)"
        echo "  - GCP_PROJECT_ID"
        echo "  - GCP_REGION"
        echo "  - GKE_CLUSTER_NAME"
        echo "Skipping GCP and GKE login."
        return 1
    fi

    # Check if service account key file exists
    check_file_exists "$GCP_SERVICE_ACCOUNT_KEY"

    echo "🔹 Authenticating with Google Cloud..."
    gcloud auth activate-service-account --key-file="$GCP_SERVICE_ACCOUNT_KEY"
    if [ $? -eq 0 ]; then
        echo "✅ Successfully authenticated with GCP."
    else
        echo "❌ Failed to authenticate with GCP."
        return 1
    fi

    # Set current project
    echo "🔹 Setting GCP project: $GCP_PROJECT_ID..."
    gcloud config set project "$GCP_PROJECT_ID"

    # GKE Cluster Login
    echo "🔹 Getting GKE credentials for cluster: $GKE_CLUSTER_NAME..."
    gcloud container clusters get-credentials "$GKE_CLUSTER_NAME" --region "$GCP_REGION" --project "$GCP_PROJECT_ID"
    if [ $? -eq 0 ]; then
        echo "✅ Successfully configured GKE cluster access."
    else
        echo "❌ Failed to configure GKE cluster access."
        return 1
    fi
}

# Perform GCP and GKE login only if PROVIDER=gcp
if [[ "$PROVIDER" == "GCP" ]]; then
    echo "🔹 GCP authentication is enabled. Performing GCP login..."
    gcp_gke_login
else
    echo "⚠️ GCP authentication is disabled. Skipping GCP login."
fi

# Docker login if credentials are provided
if [[ -n "$DOCKER_USERNAME" && -n "$DOCKER_PASSWORD" ]]; then
    echo "$DOCKER_PASSWORD" | docker login --username "$DOCKER_USERNAME" --password-stdin
    echo "✅ Successfully logged into Docker Hub."
fi

# Perform AWS and EKS login only if PROVIDER=aws
if [[ "$PROVIDER" == "aws" ]]; then
    echo "🔹 AWS authentication is enabled. Performing AWS login..."
    aws_eks_login
else
    echo "⚠️ AWS authentication is disabled. Skipping AWS login."
fi

# Initialize command with base command
SMURF_CMD="/usr/local/bin/smurf"

# Add tool as first argument if provided
if [[ $# -gt 0 ]]; then
    SMURF_CMD="$SMURF_CMD $1"
    shift
fi

# Add remaining arguments
if [[ $# -gt 0 ]]; then
    SMURF_CMD="$SMURF_CMD $*"
fi

# Debug output
echo "🔹 Executing command: $SMURF_CMD"

# Execute the final command
exec $SMURF_CMD
