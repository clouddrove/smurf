# Terraform User Guide


### Using Smurf STF on local
Use `smurf stf <command>` to run Terraform commands. Supported commands include:

- **Help:** `smurf stf --help`
- **Initialize Terraform:** `smurf stf init`
- **Generate and Show Execution Plan:** `smurf stf plan`
- **Apply Terraform Changes:** `smurf stf apply`
- **Detect Drift in Terraform State:** `smurf stf drift`
- **Provision Terraform Environment:** `smurf stf provision`

- The `provision` command for Terraform performs `init`, `validate`, and `apply`.

## Using Smurf STF in GitHub Action
 This GitHub Action Initialize Terraform and Validate Terraform changes.

```yaml
name: Smurf STF Workflow

on:
  push:
    branches:
      - master

jobs:
  terraform:
    runs-on: ubuntu-latest
    steps:
      - name: Checkout repository
        uses: actions/checkout@v3

      - name: Smurf stf init
        uses: clouddrove/smurf@master
        with:
          path:   # we can specify the path of folder where main.tf is located.
          tool: stf
          command: init

      - name: Smurf stf validate
        uses: clouddrove/smurf@master
        with:
          path:   
          tool: stf
          command: validate
```
### All available commands in Smurf STF
| Command   | Description                          |
|-----------|--------------------------------------|
| `apply`    | Apply the changes required to reach the desired state of Terraform Infrastructure |
| `destroy` | Destroy the Terraform Infrastructure |
| `drift`    | Detect drift between state and infrastructure  for Terraform  |
| `format`   | Format the Terraform Infrastructure              |
| `init` | Initialize Terraform             |
| `output` | Generate output for the current state of Terraform Infrastructure  |
| `plan` | Generate and show an execution plan for Terraform          |
| `provision` | Its the combination of init, plan, apply, output for Terraform |
| `validate` | Validate  Terraform changes |


### Available Flags for Provision Command

| Flag                  | Description                                                    | Default Value |
|-----------------------|----------------------------------------------------------------|--------------|
| `--approve`          | Skip interactive approval of plan before applying             | `true`       |
| `-h, --help`         | Display help for the provision command                        | N/A          |
| `--var string`       | Specify a variable in 'NAME=VALUE' format                     | N/A          |
| `--var-file string`  | Specify a file containing variables                           | N/A          |



