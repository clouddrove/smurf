# 
![Banner](https://github.com/clouddrove/terraform-module-template/assets/119565952/67a8a1af-2eb7-40b7-ae07-c94cde9ce062)
<h1 align="center">
    Smurf
</h1>

<p align="center">
    <a href="https://goreportcard.com/report/github.com/clouddrove/smurf">
        <img alt="Go Report Status" src="https://goreportcard.com/badge/github.com/clouddrove/smurf">
    </a>
    <a href="https://github.com/clouddrove/smurf/actions/workflows/build.yml">
        <img alt="Build Status" src="https://github.com/clouddrove/smurf/actions/workflows/build.yml/badge.svg">
    </a>
    <a href="https://www.launchpass.com/devops-talks">
        <img alt="Slack Chat" src="https://img.shields.io/badge/join%20slack-blue">
    </a>
  <a href="http://www.apache.org/licenses/LICENSE-2.0">
    <img alt="Apache-2.0 License" src="https://img.shields.io/badge/apache-2-0.svg">
  </a>
</p>


<p align="center">
<a href='https://facebook.com/sharer/sharer.php?u=https://github.com/clouddrove/smurf'>
  <img title="Share on Facebook" src="https://user-images.githubusercontent.com/50652676/62817743-4f64cb80-bb59-11e9-90c7-b057252ded50.png" />
</a>
<a href='https://www.linkedin.com/shareArticle?mini=true&title=smurf&url=https://github.com/clouddrove/smurf'>
  <img title="Share on LinkedIn" src="https://user-images.githubusercontent.com/50652676/62817742-4e339e80-bb59-11e9-87b9-a1f68cae1049.png" />
</a>
<a href='https://twitter.com/intent/tweet/?text=smurf&url=https://github.com/clouddrove/smurf'>
  <img title="Share on Twitter" src="https://user-images.githubusercontent.com/50652676/62817740-4c69db00-bb59-11e9-8a79-3580fbbf6d5c.png" />
</a>
</p>

<p align="center">
Smurf is a Go CLI that wraps Docker, Helm, and Terraform behind one consistent interface, using each tool's native Go SDK instead of shelling out. One binary, one config file, unified commands: build and push images to any major registry, install and upgrade Helm releases, plan and apply Terraform, or chain the whole pipeline with a single <code>smurf deploy</code>. Less context switching, fewer one-off scripts, the same commands locally and in CI.
</p>

## Installation ⚙️

### Homebrew (macOS and Linux)
```bash
brew tap clouddrove/homebrew-tap
brew install smurf
```

### Install script (Linux and macOS)
Downloads the release archive for your OS/arch and verifies it against `checksums.txt`:
```bash
curl -fsSL https://raw.githubusercontent.com/clouddrove/smurf/master/install/install.sh | bash
```

### Docker
The image bundles smurf together with docker-cli, aws-cli, gcloud, terraform, and trivy:
```bash
docker run --rm ghcr.io/clouddrove/smurf:latest version
```

### Go
```bash
go install github.com/clouddrove/smurf@latest
```

### Manual download
Grab the archive for your platform from the [releases page](https://github.com/clouddrove/smurf/releases), verify it against `checksums.txt`, and put the binary on your `PATH`.

### GitHub Actions
```yaml
    - name: Setup Smurf
      uses: clouddrove/smurf@v1.1.5
```

### From source
See the [installation guide](docs/sm/docs/installation.md) for building from source and platform notes.

## Quickstart 🏁

```bash
# Scaffold a smurf.yaml with both sdkr and selm sections (0600, refuses to overwrite)
smurf init

# Build the Docker image described in smurf.yaml
smurf sdkr build

# Build, push, and (if selm.deployHelm is true) install/upgrade the Helm release, all from smurf.yaml
smurf deploy
```

## Features 🚀

### 🐳 Docker Command Wrapper (`sdkr`)
Streamline Docker image workflows:
- `build`, `scan`, `tag`, `push`, `remove`, `init`
- `provision-acr`/`provision-ecr`/`provision-gcp`/`provision-ghcr`/`provision-hub` → each runs (`build` ➝ `push`), prompting `Proceed with push? [y/N]` on a TTY unless `--yes` is passed
- [Docker with Smurf – Usage Guide](docs/sdkr/README.md)

---

### ⚓ Helm Command Wrapper (`selm`)
Simplify Helm operations:
- `create`, `install`, `lint`, `list`, `status`, `template`, `upgrade`, `uninstall`, `init`, `debug`, `plugin`
- `provision` → runs (`install` ➝ `upgrade` ➝ `lint` ➝ `template`)
- [Helm with Smurf – Usage Guide](docs/selm/README.md)

---

### 🏗️ Terraform Command Wrapper (`stf`)
Easily manage Terraform workflows:
- `init`, `plan`, `apply`, `output`, `drift`, `validate`, `destroy`, `fmt`, `show`, `import`, `refresh`, `graph`, `state-list`, `state-rm`, `state-push`, `state-pull`
- `provision` → runs (`init` ➝ `plan` ➝ `apply` ➝ `output`); applying requires `--auto-approve` (default `false`)
- [Terraform with Smurf – Usage Guide](docs/stf/README.md)

---

### 🚀 `smurf deploy` command
Reads `smurf.yaml`, builds the Docker image, pushes it to whichever registry is enabled, and (if `selm.deployHelm` is true) installs or upgrades the Helm release.
- `deploy` → runs (`build` ➝ `push` ➝ Helm install/upgrade), controlled by `--timeout` (seconds, default `600`)

---

## 🧰 Credential Fallback from `smurf.yaml`

Smurf supports **automatic credential fallback**.  
If required credentials (like username or token) are not provided via CLI or environment variables, Smurf will read them directly from your `smurf.yaml` file. Values also support `${ENV_VAR}` interpolation.

See the [full `smurf.yaml` field reference](docs/sm/docs/configuration.md) for every supported key.

### Example
```yaml
sdkr:
  awsECR: false
  dockerHub: false
  ghcrRepo: false
  dockerfile: ""
  imageName: ""
selm:
  chartName: ""
  deployHelm: true
  fileName: ""
  namespace: ""
  releaseName: ""
  revision: 0
  ```

---

### ☁️ Multicloud Container Registry
Push images to multiple registries from one CLI:
- Supported: **AWS ECR**, **GCP GCR**, **Azure ACR**, **Docker Hub**
- Example:
  ```bash
  smurf sdkr push --help
  ```


## Contributors ✨ 

Big thanks to our contributors for elevating our project with their dedication and expertise! But, we do not wish to stop there, would like to invite contributions from the community in improving these projects and making them more versatile for better reach. Remember, every bit of contribution is immensely valuable, as, together, we are moving in only 1 direction, i.e. forward.

<a href="https://github.com/clouddrove/smurf/graphs/contributors">
  <img src="https://contrib.rocks/image?repo=clouddrove/smurf" />
</a>
<br>
<br> 

Want to contribute? Read [CONTRIBUTING.md](CONTRIBUTING.md) for the development setup, project layout, and pull request conventions, and [SECURITY.md](SECURITY.md) for how to report vulnerabilities privately.

## License

This project is licensed under the Apache License 2.0 - see the [LICENSE](LICENSE) file for details.

## Feedback
Spot a bug or have thoughts to share with us? Let's squash it together! Log it in our [issue tracker](https://github.com/clouddrove/smurf/issues), feel free to drop us an email at [hello@clouddrove.com](mailto:hello@clouddrove.com).

Show some love with a ★ on [our GitHub](https://github.com/clouddrove/smurf)!  if our work has brightened your day! – your feedback fuels our journey!

## Join Our Slack Community

Join our vibrant open-source slack community and embark on an ever-evolving journey with CloudDrove; helping you in moving upwards in your career path.
Join our vibrant Open Source Slack Community and embark on a learning journey with CloudDrove. Grow with us in the world of DevOps and set your career on a path of consistency.

🌐💬What you'll get after joining this Slack community:

- 🚀 Encouragement to upgrade your best version.
- 🌈 Learning companionship with our DevOps squad.
- 🌱 Relentless growth with daily updates on new advancements in technologies.

Join our tech elites [Join Now][slack] 🚀



## Tap into our capabilities
We provide a platform for organizations to engage with experienced top-tier DevOps & Cloud services. Tap into our pool of certified engineers and architects to elevate your DevOps and Cloud Solutions.

At [CloudDrove][website], has extensive experience in designing, building & migrating environments, securing, consulting, monitoring, optimizing, automating, and maintaining complex and large modern systems. With remarkable client footprints in American & European corridors, our certified architects & engineers are ready to serve you as per your requirements & schedule. Write to us at [business@clouddrove.com](mailto:business@clouddrove.com).

<p align="center">We are <b> The Cloud Experts!</b></p>
<hr />
<p align="center">We ❤️  <a href="https://github.com/clouddrove">Open Source</a> and you can check out <a href="https://registry.terraform.io/namespaces/clouddrove">our other modules</a> to get help with your new Cloud ideas.</p>

[website]: https://clouddrove.com
[blog]: https://blog.clouddrove.com
[slack]: https://www.launchpass.com/devops-talks
[github]: https://github.com/clouddrove
[linkedin]: https://linkedin.com/company/clouddrove
[twitter]: https://twitter.com/clouddrove/
[email]: https://clouddrove.com/contact-us.html
[terraform_modules]: https://github.com/clouddrove?utf8=%E2%9C%93&q=terraform-&type=&language=
