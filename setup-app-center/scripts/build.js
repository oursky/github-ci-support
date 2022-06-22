const esbuild = require("esbuild");
const { builtinModules } = require("module");
const { join } = require("path");

async function main() {
  const deps = [...builtinModules];
  await esbuild.build({
    entryPoints: [join(__dirname, "..", "src", "main.ts")],
    bundle: true,
    outfile: join(__dirname, "..", "dist", "main.js"),
    sourcemap: true,

    external: deps,
    platform: "node",
    target: "node16",
  });
}

main().catch((err) => {
  console.error(err);
  process.exit(1);
});
