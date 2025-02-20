# Installation Guide

## Prerequisites

- Go 1.20 or higher
- Git
- Terraform, Helm, and Docker Daemon installed and accessible via your PATH

## CLI Installation

### 1. **Clone the repository:**

   ```bash
   git clone https://github.com/clouddrove/smurf.git
   ```


### 2. Run the Installation Script

Execute the following command to build and install Smurf:

```bash
bash install_smurf.sh
```

### 3. Verify the Installation

To check if Smurf is installed successfully, run:

```bash
which smurf
smurf --help
```

If the output shows `/usr/local/bin/smurf` and the help menu, the installation was successful!

---

## GitHub Action Setup

```yaml
name: GitHub Action Setup

on:
  push:
    branches:
      - master
  workflow_dispatch:

jobs:
  docker-build-publish:
    runs-on: ubuntu-latest
    permissions: write-all
    
    env:
      DOCKER_USERNAME: ${{ secrets.DOCKERHUB_USERNAME }}
      DOCKER_PASSWORD: ${{ secrets.DOCKERHUB_TOKEN }}
     
    steps:
      - name: Checkout code
        uses: actions/checkout@v4.1.7
 
      - name: Dummy Step
        uses: clouddrove/smurf@v1.0.0
        with:
          tool: <tool name>
          command: <command to run>
```