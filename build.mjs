#!/usr/bin/env node

import fs from "node:fs/promises";
import { constants as fsConstants } from "node:fs";
import path from "node:path";
import process from "node:process";
import { spawnSync } from "node:child_process";
import { fileURLToPath } from "node:url";

const VERSION = "0.0.1";
const TARGETS = [
  "linux/amd64",
  "darwin/amd64",
  "darwin/arm64",
  "windows/amd64",
];

const COLORS = {
  red: "\x1b[31m",
  green: "\x1b[32m",
  yellow: "\x1b[33m",
  reset: "\x1b[0m",
};

const __filename = fileURLToPath(import.meta.url);
const projectRoot = path.resolve(path.dirname(__filename));
const webDir = path.join(projectRoot, "web");
const embedDir = path.join(projectRoot, "internal", "webserver", "static", "dist");
const outputDir = path.join(projectRoot, "build");
const frontendDistDir = path.join(webDir, "dist");

function colorize(color, text) {
  if (!process.stdout.isTTY) {
    return text;
  }
  return `${COLORS[color]}${text}${COLORS.reset}`;
}

function runCommand(command, args, options = {}) {
  const result = spawnSync(command, args, {
    stdio: "inherit",
    shell: false,
    ...options,
  });

  if (result.error) {
    throw result.error;
  }

  if (result.status !== 0) {
    throw new Error(`${command} exited with code ${result.status ?? "unknown"}`);
  }
}

function commandExists(command) {
  const result = spawnSync(command, ["--version"], {
    stdio: "ignore",
    shell: false,
  });

  if (result.error) {
    return false;
  }

  return result.status === 0;
}

async function pathExists(filePath) {
  try {
    await fs.access(filePath, fsConstants.F_OK);
    return true;
  } catch {
    return false;
  }
}

function formatBytes(bytes) {
  const units = ["B", "KB", "MB", "GB"];
  let value = bytes;
  let index = 0;

  while (value >= 1024 && index < units.length - 1) {
    value /= 1024;
    index += 1;
  }

  return `${value.toFixed(value >= 10 || index === 0 ? 0 : 1)}${units[index]}`;
}

async function ensureFrontendCopied() {
  await fs.rm(embedDir, { recursive: true, force: true });
  await fs.mkdir(embedDir, { recursive: true });
  const entries = await fs.readdir(frontendDistDir);
  for (const entry of entries) {
    await fs.cp(path.join(frontendDistDir, entry), path.join(embedDir, entry), { recursive: true });
  }
}

async function main() {
  console.log("=======================================");
  console.log("  SpoutMC Multi-Architecture Build");
  console.log("=======================================");
  console.log("");

  console.log(colorize("yellow", "[1/6] Cleaning previous builds..."));
  await fs.rm(outputDir, { recursive: true, force: true });
  await fs.mkdir(outputDir, { recursive: true });
  await fs.rm(embedDir, { recursive: true, force: true });

  console.log(colorize("yellow", "[2/6] Installing frontend dependencies..."));
  const nodeModulesDir = path.join(webDir, "node_modules");
  if (!(await pathExists(nodeModulesDir))) {
    runCommand("npm", ["install"], { cwd: webDir });
  } else {
    console.log("  -> Dependencies already installed");
  }

  console.log(colorize("yellow", "[3/6] Building frontend (Vite)..."));
  runCommand("npm", ["run", "build"], { cwd: webDir });

  if (!(await pathExists(frontendDistDir))) {
    console.error(colorize("red", "x Frontend build failed - dist directory not found"));
    process.exit(1);
  }
  console.log(colorize("green", "v Frontend build complete"));

  console.log(colorize("yellow", "[4/6] Copying frontend to embed directory..."));
  await ensureFrontendCopied();
  console.log(colorize("green", `v Frontend copied to ${embedDir}`));

  console.log(colorize("yellow", "[5/6] Generating Swagger documentation..."));
  if (commandExists("swag")) {
    runCommand("swag", ["init", "-g", "cmd/spoutmc/main.go", "--parseDependency", "--parseInternal"], {
      cwd: projectRoot,
    });
    console.log(colorize("green", "v Swagger docs generated"));
  } else {
    console.log(colorize("yellow", "! swag not found, skipping Swagger generation"));
  }

  console.log(colorize("yellow", "[6/6] Building Go binaries for multiple architectures..."));
  for (const target of TARGETS) {
    const [goos, goarch] = target.split("/");
    let outputName = `spoutmc-${goos}-${goarch}`;
    if (goos === "windows") {
      outputName += ".exe";
    }

    console.log(`  Building for ${goos}/${goarch}...`);
    runCommand(
      "go",
      [
        "build",
        `-ldflags=-s -w -X main.Version=${VERSION}`,
        "-o",
        path.join(outputDir, outputName),
        "./cmd/spoutmc",
      ],
      {
        cwd: projectRoot,
        env: {
          ...process.env,
          GOOS: goos,
          GOARCH: goarch,
        },
      },
    );

    const stat = await fs.stat(path.join(outputDir, outputName));
    console.log(colorize("green", `  v Built ${outputName} (${formatBytes(stat.size)})`));
  }

  console.log("");
  console.log(colorize("green", "======================================="));
  console.log(colorize("green", "  Build Complete!"));
  console.log(colorize("green", "======================================="));
  console.log("");
  console.log("Built binaries:");

  const outputFiles = await fs.readdir(outputDir);
  for (const file of outputFiles.sort()) {
    const stat = await fs.stat(path.join(outputDir, file));
    console.log(`  ${file} (${formatBytes(stat.size)})`);
  }

  console.log("");
  console.log("To run:");
  console.log("  Linux:   ./build/spoutmc-linux-amd64");
  console.log("  macOS:   ./build/spoutmc-darwin-amd64 (or darwin-arm64)");
  console.log("  Windows: .\\build\\spoutmc-windows-amd64.exe");
}

main().catch((error) => {
  console.error(colorize("red", `Build failed: ${error.message}`));
  process.exit(1);
});
