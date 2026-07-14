# Terraform using Smurf ⚙️

Use `smurf stf <command>` to run smurf stf commands. Supported commands include:

- **`apply`**: Apply the changes required to reach the desired state of Terraform Infrastructure.  
- **`destroy`**: Destroy the Terraform Infrastructure.  
- **`drift`**: Detect drift between state and infrastructure for Terraform.  
- **`fmt`**: Format the Terraform Infrastructure.  
- **`graph`**: Generate a visual graph of Terraform resources.  
- **`import`**: Import existing infrastructure into Terraform state.
- **`init`**: Initialize Terraform.  
- **`output`**: Generate output for the current state of Terraform Infrastructure.  
- **`plan`**: Generate and show an execution plan for Terraform.  
- **`provision`**: Combination of `init`, `plan`, `apply`, and `output` for Terraform. Applying requires `--auto-approve` (default `false`).
- **`refresh`**: Update the state file of your infrastructure.  
- **`show`**: Show Terraform state or saved plan details.
- **`state-list`**: List resources in the Terraform state.
- **`state-pull`**: Pull and display the current remote state.
- **`state-push`**: Push local state to remote backend.
- **`state-rm`**: Remove resources from the Terraform state.

## Using Smurf Terraform in local environment
Suppose you want to init, plan, apply and output for Terraform with one single command-
```bash
smurf stf provision --auto-approve
```
![stf](gif/stf_provision.mov)
