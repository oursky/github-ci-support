import * as core from "@actions/core";
import { inspect } from "util";
import { setupAppCenter } from "./index";

async function run() {
  try {
    const versionSpec = core.getInput("cli-version", {
      required: false,
    });

    await setupAppCenter(versionSpec);
  } catch (error) {
    core.info(inspect(error));
    core.setFailed(`Action failed: ${error}`);
  }
}

run();
