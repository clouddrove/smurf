#!/bin/bash
set -e

# Debugging info
# echo "Starting entrypoint script..."
# echo "Current User: $(whoami)"
# echo "Working Directory: $(pwd)"
# echo "AWS Region: ${AWS_REGION:-not set}"
# echo "EKS Cluster: ${EKS_CLUSTER_NAME:-not set}"
# echo "GCP Project: ${GCP_PROJECT_ID:-not set}"
# echo "GKE Cluster: ${GKE_CLUSTER_NAME:-not set}"

# Function to check file existence and non-emptiness
check_file_exists() {
    if [ ! -s "$1" ]; then
        echo "Error: $1 is either missing or empty."
        exit 1
    fi
}

# ✅ Function to ensure required env vars are set
require_env() {
    for var in "$@"; do
        if [ -z "${!var}" ]; then
            echo "❌ Environment variable $var is required but not set."
            exit 1
        fi
    done
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

# GCP & GKE Login
gcp_gke_login() {
  require_env GCP_PROJECT_ID GCP_REGION GKE_CLUSTER_NAME GOOGLE_APPLICATION_CREDENTIALS

  echo "🔹 Authenticating with GCP..."

  # Decode base64 GCP key if not present
  if [[ ! -f "$GOOGLE_APPLICATION_CREDENTIALS" && -n "$GCP_KEY_B64" ]]; then
    echo "$GCP_KEY_B64" | base64 -d > "$GOOGLE_APPLICATION_CREDENTIALS"
    echo "🔹 Decoded GCP key to $GOOGLE_APPLICATION_CREDENTIALS"
  fi

  if [[ ! -f "$GOOGLE_APPLICATION_CREDENTIALS" ]]; then
    echo "❌ GCP key file not found at $GOOGLE_APPLICATION_CREDENTIALS"
    exit 1
  fi

  gcloud auth activate-service-account --key-file="$GOOGLE_APPLICATION_CREDENTIALS"
  echo "🔹 Getting GKE credentials..."
  gcloud container clusters get-credentials "$GKE_CLUSTER_NAME" --region "$GCP_REGION" --project "$GCP_PROJECT_ID"
  echo "✅ GCP & GKE login complete."
}

# Docker login if credentials are provided
if [[ -n "$DOCKER_USERNAME" && -n "$DOCKER_PASSWORD" ]]; then
    echo "🔹 Logging into Docker Hub..."
    echo "$DOCKER_PASSWORD" | docker login --username "$DOCKER_USERNAME" --password-stdin
    echo "✅ Successfully logged into Docker Hub."
fi

# Authenticate based on provider
if [[ "$PROVIDER" == "aws" ]]; then
    echo "🔹 AWS authentication is enabled. Performing AWS login..."
    aws_eks_login
elif [[ "$PROVIDER" == "gcp" ]]; then
    echo "🔹 GCP authentication is enabled. Performing GCP login..."
    gcp_gke_login
# else
#     echo "⚠️ Unknown or unspecified provider: ${PROVIDER:-none}"
#     echo "⚠️ Skipping cloud provider authentication."
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