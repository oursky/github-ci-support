import * as core from "@actions/core";
import { setupFlutter } from "./index";

async function run() {
  try {
    const version = core.getInput("flutter-version", { required: true });

    await setupFlutter(version);
  } catch (error) {
    core.setFailed(`Action failed: ${error}`);
  }
}

run();
