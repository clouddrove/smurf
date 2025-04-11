# Helm using Smurf 🐳

Use `smurf selm <command>` to run smurf sdkr commands. Supported commands include:

- **`create`**: Create a new Helm chart in the specified directory.  
- **`install`**: Install a Helm chart into a Kubernetes cluster.  
- **`lint`**: Lint a Helm chart.  
- **`list`**: List all Helm releases.  
- **`provision`**: Combination of `install`, `upgrade`, `lint`, and `template` for Helm.  
- **`repo`**: Add, update, or manage chart repositories.  
- **`rollback`**: Roll back a release to a previous revision.  
- **`status`**: Status of a Helm release.  
- **`template`**: Render chart templates.  
- **`uninstall`**: Uninstall a Helm release.  
- **`upgrade`**: Upgrade a deployed Helm chart.  

## Using Smurf Helm in local environment
To upgrade a helm chart using smurf run the command-
```bash
smurf selm  upgrade smurf ./smurf -n smurf
```
![selm](gif/selm_upgrade.mov)

## Using Smurf Helm in GitHub Actions
Using Smurf Helm in GitHub Actions involves calling the Smurf shared workflow.
To lint, template and deploy helm chart workflow will look like-
```yaml
jobs: 
  dev:
   uses: clouddrove/github-shared-workflows/.github/workflows/smurf.yml@master
   with:
     aws_auth: true
     eks-cluster: <cluster_name>
     aws-role: <aws_role>
     aws-region: <aws_region>
     aws_auth_method: oidc
     helm_enable: false
     helm-lint-command: lint <helm_chart_path>
     helm-template-command: template <release_name> <helm_chart_path>
     helm_deploy_command: upgrade <release_name> --install --atomic -f <helm_chart_path>/values.yaml <helm_chart_path>
```

![selm](gif/selm.mov)