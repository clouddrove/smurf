#!/bin/sh
set -e

# Function to check file existence and non-emptiness
check_file_exists() {
if [ ! -s "$1" ]; then
echo "Error: $1 is either missing or empty in the specified path (${INPUT_PATH})."
return 1
fi
}

# Check if INPUT_PATH is set and change directory if it exists
if [ -n "${INPUT_PATH}" ]; then
cd "${INPUT_PATH}" || exit 1
# Check files if they exist
if [ -f "main.tf" ]; then
check_file_exists "main.tf" || exit 1
fi
if [ -f "Dockerfile" ]; then
check_file_exists "Dockerfile" || exit 1
fi
fi

# Docker login if credentials are provided
if [ -n "$DOCKER_USERNAME" ] && [ -n "$DOCKER_PASSWORD" ]; then
echo "$DOCKER_PASSWORD" | docker login --username "$DOCKER_USERNAME" --password-stdin
echo "Successfully logged into Docker Hub."
fi

# Initialize command with base command
SMURF_CMD="/usr/local/bin/smurf"

# Add tool as first argument
if [ $# -gt 0 ]; then
    TOOL="$1"
    shift
    SMURF_CMD="$SMURF_CMD $TOOL"
fi

# Add command and all remaining arguments as a single block
if [ $# -gt 0 ]; then
    SMURF_CMD="$SMURF_CMD $*"
fi

# Debug output
echo "Executing command: $SMURF_CMD"

# Execute the final command
exec $SMURF_CMD
