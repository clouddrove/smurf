# Installation Guide

### Prerequisites

- Go 1.20 or higher
- Git
- Terraform, Helm, and Docker Daemon installed and accessible via your PATH

### Installation Steps

1. **Clone the repository:**

   ```bash
   git clone https://github.com/clouddrove/smurf.git
   ```

1. **Change to the project directory:**

   ```bash
   cd smurf
   ```

1. **Build the tool:**

   ```bash
   go build -o smurf .
   ```

   This will build `smurf` in your project directory.