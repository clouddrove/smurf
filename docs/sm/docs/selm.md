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
- **`history`**: Prints historical revisions for a given release.
- **`pull`**: Downloads a chart from a repository
- **`init`**: Create `smurf.yaml` configuration file
- **`plugin`**: Manage plugins, which are add-on tools that extend Helm's core functionality.

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
  build-and-push:
    runs-on: ubuntu-latest
    steps:
      - name: Set up smurf
        uses: clouddrove/smurf@master
        with:
          version: latest

      - name: Helm lint
        run: |
          smurf selm lint <helm_chart_path>

      - name: Helm template
        run: |
          smurf selm template <release_name> <helm_chart_path>

      - name: Helm deploy
        run: |
          smurf selm upgrade <release_name> --install --atomic -f <helm_chart_path>/values.yaml <helm_chart_path>
```

## Using smurf.yaml configure file for Smurf Helm
Use the smurf.yaml configuration file to perform Smurf Helm both locally and in GitHub Actions.
```bash
smurf selm init
```
This creates the `smurf.yaml` configuration file for Helm.
```yaml
selm:
  releaseName: "Release Name"
  namespace: "Name Space"
  chartName: "Chart Name"
  revision: 0
```

Once complete `smurf.yaml` file then install, upgrade, lint, and template in one workflow will look like-
```yaml
jobs:
  build-and-push:
    runs-on: ubuntu-latest
    steps:
      - name: Set up smurf
        uses: clouddrove/smurf@master
        with:
          version: latest

      - name: Helm lint
        run: |
          smurf selm provision
```

![selm](gif/selm.mov)