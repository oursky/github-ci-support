import * as core from "@actions/core";
import * as tc from "@actions/tool-cache";
import * as exec from "@actions/exec";
import * as io from "@actions/io";
import * as glob from "@actions/glob";
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
  const sdkManager = await locateSDKManager(androidHome);
  core.info(`SDK Manager: ${sdkManager}`);

  await acceptLicenses(sdkManager, accept);

  const packageList = packages.split(" ").filter((x) => x.length > 0);
  await restorePackages(androidHome, packageList);
  await installPackages(sdkManager, packageList);
  await cachePackages(androidHome, packageList);
}

async function locateSDKManager(androidHome: string) {
  const globber = await glob.create(
    `${androidHome}/cmdline-tools/*/bin/sdkmanager`
  );
  let sdkManager = "";
  for await (const p of globber.globGenerator()) {
    sdkManager = p;
    break;
  }
  if (sdkManager.length === 0) {
    throw new Error("SDK manager not found in ANDROID_HOME");
  }
  return sdkManager;
}

async function acceptLicenses(sdkManager: string, accept: string) {
  if (accept.toLowerCase() !== "y") {
    core.error('Must pass "y"/"Y" to accept-licenses input');
    return;
  }

  await core.group("Accepting licenses...", async () => {
    const stdin = Buffer.from((accept + "\n").repeat(100), "utf8");
    await exec.exec(sdkManager, ["--licenses"], { input: stdin });
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

async function installPackages(sdkManager: string, packages: string[]) {
  await core.group("Installing packages...", async () => {
    // Installing packages with same dependency would double-install the dependency.
    // Ignore that since it's mostly one-off extra downloads.
    await exec.exec(sdkManager, packages);
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
