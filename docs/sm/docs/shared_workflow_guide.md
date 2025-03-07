# Smurf Integration GitHub Actions
Smurf has a [shared workflow](https://github.com/clouddrove/github-shared-workflows/tree/feat/smurf_shared_workflow/docs)  created to easily work with smurf in GitHub Actions.Shared workflows are a way to reuse things by defining them once and using them at different places.

## Example of using shared workflow for Smurf Docker
This workflow will build, scan and push docker images to the registry you want.
```yaml
jobs:
  dev:
    uses: clouddrove/github-shared-workflows/.github/workflows/smurf.yml@master
    with:
      docker_enable: true
      docker_build_command: build <img_name>:<tag>
      branch: <branch_name>
      docker_tag_command: tag <img_name>:<tag>
      docker_push: true
      docker_push_command: push <registry> <img_name>:<tag>
      image-name: <img_name>
      image-tag: <tag>
      image-tar: <img_name>.tar
      docker_scan: true
      docker_scan_command: scan <img_name>:<tag>
```

## Example of using shared workflow for Smurf Helm
This workflow will lint, template and deploy helm charts to your kubernetes cluster you want.
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