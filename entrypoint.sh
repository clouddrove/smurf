#!/bin/bash
set -e

check_file_exists() {
    if [ ! -s "$1" ]; then
        echo "❌ Error: $1 is either missing or empty."
        exit 1
    fi
}

aws_eks_login() {
    if [[ "$PROVIDER" != "aws" || "$TOOL" != "helm" ]]; then
        return 0
    fi

    if [[ -z "$AWS_ACCESS_KEY_ID" || -z "$AWS_SECRET_ACCESS_KEY" || -z "$AWS_DEFAULT_REGION" || -z "$EKS_CLUSTER_NAME" ]]; then
        echo "❌ AWS credentials or EKS cluster name missing!"
        return 1
    fi
    
    aws configure set aws_access_key_id "$AWS_ACCESS_KEY_ID"
    aws configure set aws_secret_access_key "$AWS_SECRET_ACCESS_KEY"
    aws configure set region "$AWS_DEFAULT_REGION"

    if aws eks update-kubeconfig --region "$AWS_DEFAULT_REGION" --name "$EKS_CLUSTER_NAME"; then
        echo "✅ Successfully updated kubeconfig for EKS."
        return 0
    else
        echo "❌ Failed to update kubeconfig for EKS."
        return 1
    fi
}

aws_login() {
    if [[ "$PROVIDER" != "aws" || "$TOOL" != "terraform" ]]; then
        return 0
    fi

    if [[ -z "$AWS_ACCESS_KEY_ID" || -z "$AWS_SECRET_ACCESS_KEY" || -z "$AWS_DEFAULT_REGION" ]]; then
        echo "❌ AWS credentials missing!"
        return 1
    fi
    
    aws configure set aws_access_key_id "$AWS_ACCESS_KEY_ID"
    aws configure set aws_secret_access_key "$AWS_SECRET_ACCESS_KEY"
    aws configure set region "$AWS_DEFAULT_REGION"
}

# AWS Login for EKS if needed
if [[ "$PROVIDER" == "aws" && "$TOOL" == "helm" ]]; then
    aws_eks_login
fi

# AWS Login for Terraform if needed
if [[ "$PROVIDER" == "aws" && "$TOOL" == "terraform" ]]; then
    aws_login
fi

# Docker Login
if [[ -n "$DOCKER_USERNAME" && -n "$DOCKER_PASSWORD" ]]; then
    echo "$DOCKER_PASSWORD" | docker login --username "$DOCKER_USERNAME" --password-stdin -q
    echo "✅ Successfully logged into Docker Hub."
fi

# Executing Smurf Command
SMURF_CMD="/usr/local/bin/smurf"

if [[ $# -gt 0 ]]; then
    SMURF_CMD="$SMURF_CMD $1"
    shift
fi

if [[ $# -gt 0 ]]; then
    SMURF_CMD="$SMURF_CMD $*"
fi

echo "🔹 Executing command: $SMURF_CMD"
exec $SMURF_CMD