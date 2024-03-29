import * as core from "@actions/core";
import * as exec from "@actions/exec";
import * as glob from "@actions/glob";

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
  await installPackages(sdkManager, packageList);
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

async function installPackages(sdkManager: string, packages: string[]) {
  await core.group("Installing packages...", async () => {
    // Install packages one-by-one,
    // ignore any errors, since it would retry when gradle builds.
    for (const pkg of packages) {
      try {
        await exec.exec(sdkManager, [pkg]);
      } catch {}
    }
  });
}
