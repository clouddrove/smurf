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
    <a href="https://join.slack.com/t/devops-talks/shared_invite/zt-2s2rnal1e-bRStDKSyRC~dpXA~PaJ7vQ">
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

Smurf is a command-line interface (CLI) application built using Golang leveraging technology specific SDKs, designed to simplify and automate commands for essential tools like Docker, Helm and Terraform. It provides intuitive, unified commands to execute Helm package manager, Terraform plans and Docker container management, and other DevOps tasks seamlessly from one interface. Whether you need to spin up environments, manage containers, or apply infrastructure as code, this CLI streamlines multi-tool operations, boosting productivity and reducing context-switching.

Smurf isn’t just another CLI tool—it’s your DevOps powerhouse. Managing Helm, Docker and Terraform and other essential tools separately is a hassle. Constant context-switching slows you down, and remembering multiple CLI syntaxes is frustrating. Smurf simplifies this by providing a unified, intuitive interface to streamline your workflows.
With Smurf, you can spin up environments, manage containers, and apply infrastructure as code—all from a single command-line tool. It boosts productivity, reduces errors, and helps you focus on delivering solutions rather than troubleshooting commands. If efficiency and automation matter to you, Smurf is the tool you’ve been waiting for.

## What can you do with Smurf

Docker Commands in Smurf 🐳

- **`build`**: Builds a Docker image with the specified **name** and **tag**.  
- **`provision-acr`**: Builds and pushes a Docker image to **Azure Container Registry (ACR)**.  
- **`provision-ec`**: Builds and pushes a Docker image to **AWS Elastic Container Registry (ECR)**.  
- **`provision-gcr`**: Builds and pushes a Docker image to **Google Container Registry (GCR)**.  
- **`provision-hub`**: Builds, scans, and pushes a Docker image to **Docker Hub** for enhanced security.  
- **`push`**: Pushes Docker images to **ACR, ECR, GCR,** or **Docker Hub** in one simple command.  
- **`remove`**: Deletes a Docker image from your **local system** to free up space.  
- **`scan`**: Analyzes a Docker image for known **security vulnerabilities** before deployment.  
- **`tag`**: Tags a Docker image for easy **identification** and **repository management**.   

Helm Commands in Smurf ⎈

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

Terraform Commands in Smurf ⚙️

- **`apply`**: Apply the changes required to reach the desired state of Terraform Infrastructure.  
- **`destroy`**: Destroy the Terraform Infrastructure.  
- **`drift`**: Detect drift between state and infrastructure for Terraform.  
- **`format`**: Format the Terraform Infrastructure.  
- **`graph`**: Generate a visual graph of Terraform resources.  
- **`init`**: Initialize Terraform.  
- **`output`**: Generate output for the current state of Terraform Infrastructure.  
- **`plan`**: Generate and show an execution plan for Terraform.  
- **`provision`**: Combination of `init`, `plan`, `apply`, and `output` for Terraform.  
- **`refresh`**: Update the state file of your infrastructure.  
- **`state-list`**: List resources in the Terraform state.  

Other Helping Commands in Smurf 🤝
- **`init`**: Create `smurf.yaml` configuration file 
- **`deploy`**: Build and push the Docker image, update the container details in the Helm chart’s values file, and install or upgrade the release — all in a single command.

## Contributors ✨ 

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