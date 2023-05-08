import * as core from "@actions/core";
import * as tc from "@actions/tool-cache";
import * as exec from "@actions/exec";
import * as io from "@actions/io";
import semver from "semver";
import path from "path";
import os from "os";
import { mkdtemp, writeFile } from "fs/promises";

export async function setupAppCenter(versionSpec: string) {
  const version = await resolveVersion(versionSpec);
  const toolDir = await installCLI(version);

  const { exitCode, stdout } = await exec.getExecOutput("npm root", [], {
    cwd: toolDir,
  });
  if (exitCode !== 0) {
    throw new Error("Failed to get bin directory");
  }

  const binPath = path.join(stdout.trim(), ".bin");
  core.addPath(binPath);
  await exec.exec("appcenter --version");
  await exec.exec("appcenter telemetry off");
}

async function resolveVersion(versionSpec: string) {
  const pkg = `appcenter-cli@${versionSpec || "latest"}`;
  core.info(`Resolving "${pkg}"...`);

  const data = await exec.getExecOutput("npm info --json", [pkg, "version"]);
  if (data.exitCode !== 0 || data.stdout.length === 0) {
    throw new Error("Cannot resolve package version");
  }

  let versions: string | string[] = JSON.parse(data.stdout);
  if (!Array.isArray(versions)) {
    versions = [versions];
  }

  return semver.rsort(versions)[0];
}

async function isToolAvailable(tool: string) {
  const path = await io.which(tool);
  return !!path;
}

async function installCLI(version: string): Promise<string> {
  const tool = tc.find("appcenter-cli", version);
  if (tool) {
    core.info(`Found in cache @ ${tool}`);
    return tool;
  }

  const tmpDir = await mkdtemp(path.join(os.tmpdir(), "appcenter-cli-"));
  await writeFile(path.join(tmpDir, "package.json"), "{}");

  const pkg = `appcenter-cli@${version}`;
  await core.group(`Installing ${pkg}...`, async () => {
    let cmdInstall;
    if (await isToolAvailable("pnpm")) {
      cmdInstall = "pnpm install";
    } else if (await isToolAvailable("yarn")) {
      cmdInstall = "yarn add";
    } else {
      cmdInstall = "npm install";
    }

    if ((await exec.exec(cmdInstall, [pkg], { cwd: tmpDir })) !== 0) {
      throw new Error("Failed to install CLI");
    }
  });

  core.info("Adding to the cache...");
  const cachedDir = await tc.cacheDir(tmpDir, "appcenter-cli", version);
  core.info(`Successfully cached App Center CLI to ${cachedDir}`);

  return cachedDir;
}
