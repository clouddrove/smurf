# Installation Guide

## Prerequisites

- Go 1.20 or higher
- Git
- Terraform, Helm, and Docker Daemon installed and accessible via your PATH

## Installation Steps

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

## Troubleshooting

- **"go: command not found"** → Ensure Go is installed and accessible via `PATH`.
- **"permission denied"** → Run the installation script with `sudo bash install_smurf.sh`.
- **"cannot move smurf: No such file or directory"** → Ensure `go build` is successful and the binary exists in the `build` directory.

---