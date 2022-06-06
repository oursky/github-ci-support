import * as core from "@actions/core";
import { setupAndroid } from "./index";

async function run() {
  try {
    const acceptLicenses = core.getInput("accept-licenses", { required: true });
    const packages = core.getInput("packages", { required: true });

    await setupAndroid(acceptLicenses, packages);
  } catch (error) {
    core.setFailed(`Action failed: ${error}`);
  }
}

run();
