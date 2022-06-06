import * as core from "@actions/core";
import * as tc from "@actions/tool-cache";
import * as exec from "@actions/exec";
import { HttpClient } from "@actions/http-client";
import semver from "semver";
import path from "path";

interface Manifest {
  base_url: string;
  releases: Release[];
}

interface Release {
  channel: string;
  version: string;
  dart_sdk_arch: string;
  archive: string;
}

function manifestURL(os: string): string {
  return `https://storage.googleapis.com/flutter_infra_release/releases/releases_${os.toLowerCase()}.json`;
}

export async function setupFlutter(version: string) {
  const os = process.env["RUNNER_OS"] ?? "";
  const arch = process.env["RUNNER_ARCH"] ?? "";
  core.info(`OS: ${os}`);
  core.info(`Arch: ${arch}`);
  core.info(`Version: ${version}`);

  await core.group("Installing flutter...", async () => {
    const client = new HttpClient();
    const manifest = await fetchManifest(client, os);
    const release = resolveRelease(manifest, arch, version);
    const flutterDir = await fetchRelease(manifest, release);
    setupEnv(flutterDir);
  });

  await exec.exec("flutter config --no-analytics");
  await exec.exec("flutter doctor");
}

async function fetchManifest(
  client: HttpClient,
  os: string
): Promise<Manifest> {
  const url = manifestURL(os);
  core.info(`Fetching manifest: ${url}`);

  const resp = await client.getJson<Manifest>(url);

  if (resp.statusCode === 404) {
    throw new Error(`manifest for OS ${os} not found`);
  } else if (resp.statusCode === 200 && resp.result) {
    return resp.result;
  } else {
    throw new Error(`failed to download manifest for OS ${os}`);
  }
}

function resolveRelease(
  { releases }: Manifest,
  arch: string,
  version: string
): Release {
  const archReleases = releases.filter(
    (r) =>
      !r.dart_sdk_arch || r.dart_sdk_arch.toLowerCase() === arch.toLowerCase()
  );
  if (archReleases.length === 0) {
    const archs = [...new Set(releases.map((r) => r.dart_sdk_arch))].join(", ");
    throw new Error(`no release for arch ${arch}; available archs: ${archs}`);
  }

  const matchedVersion = semver.minSatisfying(
    archReleases.map((r) => r.version),
    version
  );
  if (!matchedVersion) {
    throw new Error(`no matching release for version ${version}`);
  }
  return archReleases.find((r) => r.version === matchedVersion)!;
}

async function fetchRelease({ base_url }: Manifest, release: Release) {
  const toolPath = tc.find("flutter", release.version, release.dart_sdk_arch);
  if (toolPath) {
    core.info(`Found in cache @ ${toolPath}`);
    return toolPath;
  }

  const url = [base_url, release.archive].join("/");
  core.info(`Acquiring ${release.version} from ${url}`);
  const archivePath = await tc.downloadTool(url);

  core.info("Extracting Flutter...");
  let extPath: string;
  if (url.endsWith(".zip")) {
    extPath = await tc.extractZip(archivePath);
  } else {
    extPath = await tc.extractTar(archivePath, undefined, "xJ");
  }
  extPath = path.join(extPath, "flutter");
  core.info(`Successfully extracted Flutter to ${extPath}`);

  core.info("Precaching...");
  await exec.exec(path.join(extPath, "bin", "flutter"), ["precache"]);

  core.info("Adding to the cache...");
  const cachedDir = await tc.cacheDir(
    extPath,
    "flutter",
    release.version,
    release.dart_sdk_arch
  );
  core.info(`Successfully cached Flutter to ${cachedDir}`);

  return extPath;
}

function setupEnv(flutterDir: string): void {
  core.addPath(path.join(flutterDir, "bin"));
}
