import * as core from "@actions/core";
import { setupFlutter } from "./index";

async function run() {
  try {
    const version = core.getInput("flutter-version", { required: true });
    const cache = core.getInput("flutter-cache");

    await setupFlutter(version, cache);
  } catch (error) {
    core.setFailed(`Action failed: ${error}`);
  }
}

run();
