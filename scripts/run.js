#!/usr/bin/env node

const fs = require("fs");
const path = require("path");
const { execFileSync } = require("child_process");

const binaryName = process.platform === "win32" ? "md2x.exe" : "md2x";
const binaryPath = path.join(__dirname, "..", "bin", binaryName);

if (!fs.existsSync(binaryPath)) {
  console.error(
    "md2x binary is missing. Reinstall with: npm install -g @geekjourneyx/md2x"
  );
  process.exit(1);
}

try {
  execFileSync(binaryPath, process.argv.slice(2), { stdio: "inherit" });
} catch (error) {
  if (typeof error.status === "number") {
    process.exit(error.status);
  }
  console.error(error.message);
  process.exit(1);
}
