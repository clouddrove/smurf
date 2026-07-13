### Using Smurf STF on local
Use `smurf stf <command>` to run Terraform commands. Supported commands include:

- **Help:** `smurf stf --help`
- **Initialize Terraform:** `smurf stf init`
- **Generate and Show Execution Plan:** `smurf stf plan`
- **Apply Terraform Changes:** `smurf stf apply`
- **Detect Drift in Terraform State:** `smurf stf drift`
- **Provision Terraform Environment:** `smurf stf provision`

The `provision` command for Terraform performs `init`, `plan`, `apply`, and `output`. Applying requires `--auto-approve` (default `false`); without it, `provision` stops after `plan` without touching infrastructure.

### Using Smurf STF in GitHub Action
### This GitHub Action installs Smurf, then Terraform init and validate run as regular steps.

```yaml
name: Smurf STF Workflow

on:
  push:
    branches:
      - main

jobs:
  terraform:
    runs-on: ubuntu-latest
    steps:
      - name: Checkout repository
        uses: actions/checkout@v3

      - name: Setup Smurf
        uses: clouddrove/smurf@v1.1.2

      - name: Smurf stf init
        working-directory: tf
        run: smurf stf init

      - name: Smurf stf validate
        working-directory: tf
        run: smurf stf validate
```

### All available commands in Smurf STF

| Command   | Description                          |
|-----------|--------------------------------------|
| `apply`    | Apply the changes required to reach the desired state of Terraform Infrastructure |
| `destroy` | Destroy the Terraform Infrastructure |
| `drift`    | Detect drift between state and infrastructure  for Terraform  |
| `fmt`   | Format the Terraform Infrastructure              |
| `graph` | Generate a visual graph of Terraform resources |
| `import` | Import existing infrastructure into Terraform state |
| `init` | Initialize Terraform             |
| `output` | Generate output for the current state of Terraform Infrastructure  |
| `plan` | Generate and show an execution plan for Terraform          |
| `provision` | Combination of init, plan, apply, and output for Terraform (apply requires `--auto-approve`) |
| `refresh` | Update the state file of your infrastructure |
| `show` | Show Terraform state or saved plan details |
| `state-list` | List resources in the Terraform state |
| `state-pull` | Pull and display the current remote state |
| `state-push` | Push local state to remote backend |
| `state-rm` | Remove resources from the Terraform state |
| `validate` | Validate  Terraform changes |
