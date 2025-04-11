# Installation Guide

## Prerequisites

- Go 1.20 or higher
- Git
- Terraform, Helm, and Docker Daemon installed and accessible via your PATH

## Setup using GitHub Repository

1. **Clone the repository:**

   ```bash
   git clone https://github.com/clouddrove/smurf.git
   ```
2. Navigate into the cloned project directory

```bash
cd smurf
```
3. Build the tool using Go

```bash
go build -o smurf .
```
4. Move binary to /usr/local/bin

```bash
mv smurf /usr/local/bin/
```
## Setup using Installation Script

1. **Clone the repository:**

   ```bash
   git clone https://github.com/clouddrove/smurf.git
   ```
2. Execute the following command to build and install Smurf

```bash
bash docs/sm/docs/install_smurf.sh
```
3. To check if Smurf is installed successfully, run:

```bash
which smurf
smurf --help
```

## Setup using Brew

```bash
brew tap clouddrove/homebrew-tap
brew install smurf
```

## Troubleshooting

- **"go: command not found"** → Ensure Go is installed and accessible via `PATH`.
- **"permission denied"** → Run the installation script with `sudo bash install_smurf.sh`.
- **"cannot move smurf: No such file or directory"** → Ensure `go build` is successful and the binary exists in the `build` directory.