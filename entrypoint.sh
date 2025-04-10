#!/bin/bash
set -e

# Debugging info
echo "Starting entrypoint script..."
echo "Current User: $(whoami)"
echo "Working Directory: $(pwd)"
echo "AWS Region: ${AWS_REGION:-not set}"
echo "EKS Cluster: ${EKS_CLUSTER_NAME:-not set}"
echo "GCP Project: ${GCP_PROJECT_ID:-not set}"
echo "GKE Cluster: ${GKE_CLUSTER_NAME:-not set}"

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

gcp_gke_login() {
    if [ "$INPUT_GCP_AUTH_METHOD" == "wip" ]; then
        echo "🔹 Using Workload Identity Provider authentication method"
        
        # Validate required parameters
        if [ -z "$INPUT_WORKLOAD_IDENTITY_PROVIDER" ]; then
            echo "❌ Error: workload_identity_provider is required for WIP authentication"
            exit 1
        fi
        
        if [ -z "$INPUT_SERVICE_ACCOUNT" ]; then
            echo "❌ Error: service_account is required for WIP authentication"
            exit 1
        fi
        
        if [ -z "$INPUT_GCP_PROJECT_ID" ]; then
            echo "❌ Error: gcp_project_id is required"
            exit 1
        fi
        
        # Authenticate with gcloud using workload identity
        echo "🔹 Authenticating with Google Cloud using Workload Identity Federation..."
        gcloud auth login --brief \
            --impersonate-service-account="$INPUT_SERVICE_ACCOUNT" \
            --workload-identity-provider="$INPUT_WORKLOAD_IDENTITY_PROVIDER" \
            --project="$GCP_PROJECT_ID" \
            --access-token-lifetime="300s"
        
        # Configure kubectl for GKE if cluster details are provided
        if [ ! -z "$GKE_CLUSTER_NAME" ] && [ ! -z "$GCP_REGION" ]; then
            echo "🔹 Configuring kubectl for GKE cluster: $GKE_CLUSTER_NAME"
            gcloud container clusters get-credentials "$GKE_CLUSTER_NAME" \
                --region="$GCP_REGION" \
                --project="$GCP_PROJECT_ID"
        fi
        
        echo "✅ Successfully authenticated with GCP using Workload Identity Provider"
    else
        echo "⚠️ Authentication method is not 'wip', skipping GCP authentication"
        return 1
    fi
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
else
    echo "⚠️ Unknown or unspecified provider: ${PROVIDER:-none}"
    echo "⚠️ Skipping cloud provider authentication."
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
