const fs = require("node:fs");

if (process.env.npm_lifecycle_event === process.env.VALIDATION_UPDATE_STEP) {
  const manifest = JSON.parse(fs.readFileSync("package.json", "utf8"));
  switch (process.env.VALIDATION_UPDATE_ACTION) {
    case "add":
      manifest.scripts.build = "node -e \"console.log('added build ran'); process.exit(17)\"";
      break;
    case "remove":
      delete manifest.scripts.build;
      break;
    case "invalidate":
      fs.writeFileSync("package.json", "{");
      return;
  }
  fs.writeFileSync("package.json", JSON.stringify(manifest));
}
