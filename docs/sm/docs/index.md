# 
![Banner](https://github.com/clouddrove/terraform-module-template/assets/119565952/67a8a1af-2eb7-40b7-ae07-c94cde9ce062)
<h1 align="center">
    Smurf 🚀
</h1>

Smurf is a command-line interface (CLI) application built in Golang that leverages technology-specific SDKs to simplify and automate operations for essential DevOps tools such as Docker, Helm, and Terraform. It provides intuitive, unified commands to manage Docker containers, execute Helm package operations, and apply Terraform plans—all from a single interface. Whether you need to spin up environments, manage containers, or implement infrastructure as code, Smurf streamlines multi-tool workflows, enhances productivity, and minimizes context switching.

## 🚀 Get Started  

1. **Clone the repository:**

   ```bash
   git clone https://github.com/clouddrove/smurf.git
   ```
2. Execute the following command to build and install Smurf

```bash
bash smurf/install_smurf.sh
```
3. To check if Smurf is installed successfully, run:

```bash
which smurf
smurf --help
```

## 🌟 Why Smurf?

✅ Unified Interface

Manage Docker, Helm and Terraform from a single CLI, reducing context-switching and boosting productivity.

⚡ Simplified Workflows

Automate complex operations with single-line commands.

🔒 Security Integrated

Built-in Docker image scanning before deployment ensures vulnerability-free containers.

## 🛠️ Supported Tools

🐳 Docker

Build, scan, deploy and do more with container images seamlessly.
```sh
smurf sdkr --help
```

🎩 Helm

Deploy applications effortlessly using Helm.
```sh
smurf selm --help
```

☁️ Terraform

Manage infrastructure with ease.
```sh
smurf stf --help
```
