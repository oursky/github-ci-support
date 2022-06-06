import * as core from "@actions/core";
import * as tc from "@actions/tool-cache";
import * as exec from "@actions/exec";
import * as io from "@actions/io";
import path from "path";
import fs from "fs";

const dummyVersion = "1.0.0";

function packageCacheKey(pkg: string) {
  return `android-sdk-${pkg.replaceAll(";", "-")}`;
}

export async function setupAndroid(accept: string, packages: string) {
  const androidHome = process.env["ANDROID_HOME"];
  if (!androidHome) {
    core.error("ANDROID_HOME not set");
    return;
  }
  core.info(`Android HOME: ${androidHome}`);

  await acceptLicenses(accept);

  const packageList = packages.split(" ").filter((x) => x.length > 0);
  await restorePackages(androidHome, packageList);
  await installPackages(packageList);
  await cachePackages(androidHome, packageList);
}

async function acceptLicenses(accept: string) {
  if (accept.toLowerCase() !== "y") {
    core.error('Must pass "y"/"Y" to accept-licenses input');
    return;
  }

  await core.group("Accepting licenses...", async () => {
    const stdin = Buffer.from((accept + "\n").repeat(100), "utf8");
    await exec.exec("sdkmanager", ["--licenses"], { input: stdin });
  });
}

async function restorePackages(androidHome: string, packages: string[]) {
  for (const pkg of packages) {
    const toolPath = tc.find(packageCacheKey(pkg), dummyVersion);
    if (toolPath) {
      core.info(`Found ${pkg} in cache @ ${toolPath}`);

      const packageDir = path.join(androidHome, ...pkg.split(";"));
      core.info(`Linking ${packageDir} to ${toolPath}...`);
      await io.mkdirP(path.dirname(packageDir));
      fs.symlinkSync(toolPath, packageDir, "dir");
    }
  }
}

async function installPackages(packages: string[]) {
  await core.group("Installing packages...", async () => {
    // Installing packages with same dependency would double-install the dependency.
    // Ignore that since it's mostly one-off extra downloads.
    await exec.exec("sdkmanager", packages);
  });
}

async function cachePackages(androidHome: string, packages: string[]) {
  for (const pkg of packages) {
    const toolPath = tc.find(packageCacheKey(pkg), dummyVersion);
    const packageDir = path.join(androidHome, ...pkg.split(";"));
    if (!toolPath && fs.existsSync(packageDir)) {
      core.info(`Caching ${pkg} @ ${packageDir}`);
      await tc.cacheDir(packageDir, packageCacheKey(pkg), dummyVersion);
    }
  }
}
