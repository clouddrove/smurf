#!/bin/bash
set -e

# Check if INPUT_PATH is set and non-empty
if [ -n "${INPUT_PATH}" ]; then
  cd "${INPUT_PATH}" || exit 1
  
  # Check if main.tf exists and is not empty
  if [ ! -s "main.tf" ]; then
    echo "Error: main.tf is either missing or empty in the specified path (${INPUT_PATH})."
    exit 1
  fi
fi

# Execute the smurf command
exec "/usr/local/bin/smurf" "$@"