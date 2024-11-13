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
    <a href="https://join.slack.com/t/devops-talks/shared_invite/zt-2s2rnal1e-bRStDKSyRC~dpXA~PaJ7vQ">
        <img alt="Slack Chat" src="https://img.shields.io/badge/join%20slack-click%20here-blue">
    </a>
    <a href="https://medium.com/devops-talks/announcing-devopstalks-spectacular-hacktoberfest-2024-363a09223c45">
        <img alt="Blog" src="https://img.shields.io/badge/hacktoberfest2024%20blog-8A2BE2">
    </a>
	<a href="https://choosealicense.com/licenses/mit/">
		<img alt="Apache-2.0 License" src="http://img.shields.io/badge/license-MIT-brightgreen.svg">
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




## ✨ Contributors

Big thanks to our contributors for elevating our project with their dedication and expertise! But, we do not wish to stop there, would like to invite contributions from the community in improving these projects and making them more versatile for better reach. Remember, every bit of contribution is immensely valuable, as, together, we are moving in only 1 direction, i.e. forward.

<a href="https://github.com/clouddrove/smurf/graphs/contributors">
  <img src="https://contrib.rocks/image?repo=clouddrove/smurf&max" />
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

## Explore Our Blogs

Click [here][blog] :books: :star2:

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
[linkedin]: https://cpco.io/linkedin
[twitter]: https://twitter.com/clouddrove/
[email]: https://clouddrove.com/contact-us.html
[terraform_modules]: https://github.com/clouddrove?utf8=%E2%9C%93&q=terraform-&type=&language=