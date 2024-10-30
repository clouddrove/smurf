# Smurf 

Smurf is a command-line interface built with Cobra, designed to simplify and automate commands for essential tools like Terraform and Docker. It provides intuitive, unified commands to execute Terraform plans, Docker container management, and other DevOps tasks seamlessly from one interface. Whether you need to spin up environments, manage containers, or apply infrastructure as code, this CLI streamlines multi-tool operations, boosting productivity and reducing context-switching.

## Features

- **Terraform Command Wrapper:** Run `init`, `plan`, `apply`, `output` , `drift` commands and `provision` , which is a wrapper of init, drift, plan, apply, output.
- **Git Integration:** (yet to come)
- **Docker Integration:**(yet to come)
- **Helm Integration:**(yet to come)

## Installation

### Prerequisites

- Go 1.20 or higher
- Git
- Terraform installed and available in your PATH

### Steps

1. **Clone the repository:**

   ```bash
   git clone https://github.com/clouddrove/smurf.git
   ```

2. **Change to the project directory:**

   ```bash
   cd smurf
   ```

3. **Build and install the tool:**

   ```bash
   go build .
   ```

   This will build `smurf` to your project directory.

## Usage

Navigate to your Terraform project directory and use `smurf` commands as follows:

### Initialize Terraform

```bash
./smurf init
```

This initializes your Terraform working directory.

### Generate and Show an Execution Plan for Terraform

```bash
./smurf plan
```

Generates an execution plan and shows what actions Terraform will take.

### Apply Terraform Changes

```bash
./smurf apply
```

Applies the changes required to reach the desired state of the configuration.

### Detect Drift in Terraform State

```bash
./smurf drift
```

Detects any drift between your Terraform state and the actual infrastructure.

## Important Notes

- **Uncommitted Changes:** `smurf` will check for uncommitted changes in your `.tf` files. If any are detected, it will prompt you to commit or discard them before proceeding. This ensures that only committed changes are applied, maintaining consistency and traceability.

- **Git Integration:** Make sure your project is initialized as a Git repository (`git init`) and that your `.tf` files are tracked.

## Contributing

Contributions are welcome! Please open issues or pull requests on the [GitHub repository](https://github.com/clouddrove/smurf).

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.