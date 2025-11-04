# 
![Banner](https://github.com/clouddrove/terraform-module-template/assets/119565952/67a8a1af-2eb7-40b7-ae07-c94cde9ce062)
<h1 align="center">
    Smurf
</h1>

<p align="center">
    <a href="https://goreportcard.com/report/github.com/clouddrove/smurf">
        <img alt="Go Report Status" src="https://goreportcard.com/badge/github.com/clouddrove/smurf">
    </a>
    <a href="https://github.com/clouddrove/smurf/">
        <img alt="Build Status" src="https://img.shields.io/badge/test-passing-green">
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
Smurf is a command-line interface (CLI) application built using Golang leveraging technology specific SDKs, designed to simplify and automate commands for essential tools like Terraform and Docker. It provides intuitive, unified commands to execute Terraform plans, Docker container management, and other DevOps tasks seamlessly from one interface. Whether you need to spin up environments, manage containers, or apply infrastructure as code, this CLI streamlines multi-tool operations, boosting productivity and reducing context-switching.
</p>

## Installation ⚙️
- [Smurf tool CLI Installation Guide](docs/sm/docs/installation.md)
- Integrate in GitHub Actions
  ```yaml
    - name: Setup Smurf
      uses: clouddrove/smurf@v0.0.4
  ```

## Features 🚀

### 🐳 Docker Command Wrapper (`sdkr`)
Streamline Docker image workflows:
- `build`, `scan`, `tag`, `publish`, `push`
- `provision` → runs (`build` ➝ `scan` ➝ `publish`)
- [Docker with Smurf – Usage Guide](docs/sdkr/README.md)

---

### ⚓ Helm Command Wrapper (`selm`)
Simplify Helm operations:
- `create`, `install`, `lint`, `list`, `status`, `template`, `upgrade`, `uninstall`
- `provision` → runs (`install` ➝ `upgrade` ➝ `lint` ➝ `template`)
- [Helm with Smurf – Usage Guide](docs/selm/README.md)

---

### 🏗️ Terraform Command Wrapper (`stf`)
Easily manage Terraform workflows:
- `init`, `plan`, `apply`, `output`, `drift`, `validate`, `destroy`, `format`
- `provision` → runs (`init` ➝ `validate` ➝ `apply`)
- [Terraform with Smurf – Usage Guide](docs/stf/README.md)

---

### 🚀 `smurf deploy` command
Streamline read credential from `smurf.yaml` file Docker image build, push on given repo and easily deploy using `smurf selm`
- `deploy` → runs (`build` ➝ `push` ➝ `deploy`)

---

## 🧰 Credential Fallback from `smurf.yaml`

Smurf supports **automatic credential fallback**.  
If required credentials (like username or token) are not provided via CLI or environment variables, Smurf will read them directly from your `smurf.yaml` file.

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

If you're considering contributing to our project, here are a few quick guidelines that we have been following (Got a suggestion? We are all ears!):

- **Fork the Repository:** Create a new branch for your feature or bug fix.
- **Coding Standards:** You know the drill.
- **Clear Commit Messages:** Write clear and concise commit messages to facilitate understanding.
- **Thorough Testing:** Test your changes thoroughly before submitting a pull request.
- **Documentation Updates:** Include relevant documentation updates if your changes impact it.

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

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
