#!/bin/bash
set -e

check_file_exists() {
    if [ ! -s "$1" ]; then
        echo "Error: $1 is either missing or empty."
        exit 1
    fi
}

aws_eks_login() {
    if [[ "$PROVIDER" != "aws" || "$TOOL" != "helm" ]]; then
        return 0
    fi

    if [[ -z "$AWS_ACCESS_KEY_ID" || -z "$AWS_SECRET_ACCESS_KEY" || -z "$AWS_DEFAULT_REGION" || -z "$EKS_CLUSTER_NAME" ]]; then
        return 1
    fi
    
    aws configure set aws_access_key_id "$AWS_ACCESS_KEY_ID"
    aws configure set aws_secret_access_key "$AWS_SECRET_ACCESS_KEY"
    aws configure set default.region "$AWS_DEFAULT_REGION"

    aws eks update-kubeconfig --region "$AWS_DEFAULT_REGION" --name "$EKS_CLUSTER_NAME"
    if [ $? -eq 0 ]; then
    else
        return 1
    fi
}

if [[ "$PROVIDER" == "aws" || "$TOOL" == "helm" ]]; then
        aws_eks_login
    fi
fi

aws_login() {
    if [[ "$PROVIDER" != "aws" || "$TOOL" != "terraform" ]]; then
        return 0
    fi

    if [[ -z "$AWS_ACCESS_KEY_ID" || -z "$AWS_SECRET_ACCESS_KEY" || -z "$AWS_DEFAULT_REGION" ]]; then
        return 1
    fi
    
    aws configure set aws_access_key_id "$AWS_ACCESS_KEY_ID"
    aws configure set aws_secret_access_key "$AWS_SECRET_ACCESS_KEY"
    aws configure set default.region "$AWS_DEFAULT_REGION"
}

if [[ -n "$DOCKER_USERNAME" && -n "$DOCKER_PASSWORD" ]]; then
    echo "$DOCKER_PASSWORD" | docker login --username "$DOCKER_USERNAME" --password-stdin
    echo "✅ Successfully logged into Docker Hub."
fi

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