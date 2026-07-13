# Installation Guide

## Prerequisites

- Go 1.26 or higher
- Git
- Helm, Terraform and Docker Daemon installed and accessible via your PATH

## Setup using the binary installer (recommended)

`install/install.sh` downloads the release archive for your OS/architecture (`.tar.gz` on Linux/macOS, `.zip` on Windows), verifies it against the release's `checksums.txt` by default, and installs the `smurf` binary to `/usr/local/bin`.

```bash
curl -fsSL https://raw.githubusercontent.com/clouddrove/smurf/master/install/install.sh | bash
```

To skip checksum verification (not recommended), set `VERIFY_CHECKSUM=false`:

```bash
VERIFY_CHECKSUM=false bash install/install.sh
```

The script prompts for `sudo` only when it needs to write to `/usr/local/bin`; nothing else in the script runs as root.

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
go build -a \
  -ldflags "\
    -X 'github.com/clouddrove/smurf/cmd.version=$(git describe --tags --abbrev=0 2>/dev/null || echo v0.0.0)' \
    -X 'github.com/clouddrove/smurf/cmd.commit=$(git rev-parse --short HEAD)' \
    -X 'github.com/clouddrove/smurf/cmd.date=$(date -u +%Y-%m-%dT%H:%M:%SZ)'" \
  -o smurf .
```
4. Move the binary to `/usr/local/bin` (this is the only step that needs elevated permissions)

```bash
sudo mv smurf /usr/local/bin/
```

## Setup using Installation Script

1. **Clone the repository:**

   ```bash
   git clone https://github.com/clouddrove/smurf.git
   ```
2. Execute the following command to build and install Smurf. Run it as your normal user; it calls `sudo` internally only for the final move into `/usr/local/bin`.

```bash
bash smurf/install_smurf.sh
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
- **"permission denied"** → Do not run the whole script with `sudo`; run `bash install_smurf.sh` as your normal user; it elevates only for the final `mv`/`chmod` into `/usr/local/bin`.
- **"cannot move smurf: No such file or directory"** → Ensure `go build` is successful and the binary exists in the `build` directory.
- **Checksum verification failed** → The downloaded archive does not match `checksums.txt` for that release; re-download rather than bypassing verification.