import * as core from "@actions/core";
import { setupAppCenter } from "./index";

async function run() {
  try {
    const versionSpec = core.getInput("cli-version", {
      required: false,
    });

    await setupAppCenter(versionSpec);
  } catch (error) {
    core.setFailed(`Action failed: ${error}`);
  }
}

run();
