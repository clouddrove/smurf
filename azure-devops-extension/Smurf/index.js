"use strict";

const fs = require("fs");
const https = require("https");
const os = require("os");
const path = require("path");
const { execFileSync } = require("child_process");
const task = require("azure-pipelines-task-lib/task");

const REPO = "clouddrove/smurf";

function normalizeVersion(input) {
  const version = (input || "latest").trim();
  if (version === "latest") {
    return version;
  }

  if (!/^v\d+\.\d+\.\d+$/.test(version)) {
    throw new Error("Invalid version format. Expected latest or vX.Y.Z.");
  }

  return version;
}

function resolvePlatform() {
  const platform = os.platform();
  const arch = os.arch();

  let releaseOS;
  if (platform === "linux") {
    releaseOS = "linux";
  } else if (platform === "darwin") {
    releaseOS = "darwin";
  } else if (platform === "win32") {
    releaseOS = "windows";
  } else {
    throw new Error(`Unsupported agent OS: ${platform}`);
  }

  let releaseArch;
  if (arch === "x64") {
    releaseArch = "amd64";
  } else if (arch === "arm64") {
    releaseArch = "arm64";
  } else {
    throw new Error(`Unsupported agent architecture: ${arch}`);
  }

  return { releaseOS, releaseArch, isWindows: platform === "win32" };
}

async function resolveLatestVersion() {
  const latestUrl = `https://api.github.com/repos/${REPO}/releases/latest`;
  const release = JSON.parse(await getText(latestUrl));

  if (!release.tag_name) {
    throw new Error("GitHub latest release response did not include tag_name.");
  }

  return release.tag_name;
}

async function installSmurf(versionInput) {
  const normalizedVersion = normalizeVersion(versionInput);
  const version = normalizedVersion === "latest" ? await resolveLatestVersion() : normalizedVersion;
  const { releaseOS, releaseArch, isWindows } = resolvePlatform();
  const extension = isWindows ? "zip" : "tar.gz";
  const fileName = `smurf-${version}-${releaseOS}-${releaseArch}.${extension}`;
  const url = `https://github.com/${REPO}/releases/download/${version}/${fileName}`;
  const baseTemp = task.getVariable("Agent.TempDirectory") || os.tmpdir();
  const workDir = fs.mkdtempSync(path.join(baseTemp, "smurf-"));
  const archive = path.join(workDir, fileName);
  const installDir = path.join(workDir, "bin");

  console.log(`Installing smurf ${version} for ${releaseOS}/${releaseArch}`);
  console.log(`Downloading ${url}`);

  await downloadFile(url, archive);
  fs.mkdirSync(installDir, { recursive: true });

  if (isWindows) {
    execFileSync("powershell", [
      "-NoLogo",
      "-NoProfile",
      "-Command",
      `Expand-Archive -LiteralPath ${quotePowerShell(archive)} -DestinationPath ${quotePowerShell(installDir)} -Force`,
    ]);
  } else {
    execFileSync("tar", ["-xzf", archive, "-C", installDir]);
  }

  const binaryName = isWindows ? "smurf.exe" : "smurf";
  const binaryPath = findFile(installDir, binaryName);

  if (!binaryPath) {
    throw new Error(`Downloaded archive did not contain ${binaryName}.`);
  }

  if (!isWindows) {
    fs.chmodSync(binaryPath, 0o755);
  }

  task.prependPath(path.dirname(binaryPath));
  return binaryPath;
}

function getText(url) {
  return new Promise((resolve, reject) => {
    https
      .get(url, { headers: { "User-Agent": "smurf-azure-pipelines-task" } }, (response) => {
        if (response.statusCode && response.statusCode >= 300 && response.statusCode < 400 && response.headers.location) {
          getText(response.headers.location).then(resolve, reject);
          return;
        }

        if (response.statusCode !== 200) {
          reject(new Error(`Request failed with status ${response.statusCode}: ${url}`));
          response.resume();
          return;
        }

        let body = "";
        response.setEncoding("utf8");
        response.on("data", (chunk) => {
          body += chunk;
        });
        response.on("end", () => resolve(body));
      })
      .on("error", reject);
  });
}

function downloadFile(url, destination) {
  return new Promise((resolve, reject) => {
    const file = fs.createWriteStream(destination);

    https
      .get(url, { headers: { "User-Agent": "smurf-azure-pipelines-task" } }, (response) => {
        if (response.statusCode && response.statusCode >= 300 && response.statusCode < 400 && response.headers.location) {
          file.close();
          fs.rmSync(destination, { force: true });
          downloadFile(response.headers.location, destination).then(resolve, reject);
          return;
        }

        if (response.statusCode !== 200) {
          file.close();
          fs.rmSync(destination, { force: true });
          reject(new Error(`Download failed with status ${response.statusCode}: ${url}`));
          response.resume();
          return;
        }

        response.pipe(file);
        file.on("finish", () => {
          file.close(resolve);
        });
      })
      .on("error", (err) => {
        file.close();
        fs.rmSync(destination, { force: true });
        reject(err);
      });
  });
}

function quotePowerShell(value) {
  return `'${value.replace(/'/g, "''")}'`;
}

function findFile(root, fileName) {
  for (const entry of fs.readdirSync(root, { withFileTypes: true })) {
    const fullPath = path.join(root, entry.name);
    if (entry.isFile() && entry.name === fileName) {
      return fullPath;
    }
    if (entry.isDirectory()) {
      const found = findFile(fullPath, fileName);
      if (found) {
        return found;
      }
    }
  }
  return "";
}

function parseCommand(command) {
  const input = command.trim();
  if (!input) {
    return [];
  }

  const args = [];
  let current = "";
  let quote = "";

  for (let i = 0; i < input.length; i += 1) {
    const char = input[i];

    if (quote) {
      if (char === quote) {
        quote = "";
      } else {
        current += char;
      }
      continue;
    }

    if (char === "\"" || char === "'") {
      quote = char;
      continue;
    }

    if (/\s/.test(char)) {
      if (current) {
        args.push(current);
        current = "";
      }
      continue;
    }

    current += char;
  }

  if (quote) {
    throw new Error("Command contains an unterminated quote.");
  }

  if (current) {
    args.push(current);
  }

  return args;
}

async function run() {
  try {
    const version = task.getInput("version", false) || "latest";
    const command = task.getInput("command", false) || "";
    const workingDirectory = task.getPathInput("workingDirectory", false, false);
    const smurfPath = await installSmurf(version);

    if (!command.trim()) {
      console.log("smurf installed successfully.");
      return;
    }

    const args = parseCommand(command);
    const runner = task.tool(smurfPath);
    runner.arg(args);

    const options = {};
    if (workingDirectory) {
      options.cwd = workingDirectory;
    }

    await runner.exec(options);
  } catch (err) {
    task.setResult(task.TaskResult.Failed, err.message);
  }
}

run();
