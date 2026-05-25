// Stdio launcher for Smithery MCPB bundling (runs published GHCR image).
const { spawn } = require("node:child_process");

const userID = process.env.MEMEX_USER_ID || "default";
const hybrid = process.env.MEMEX_HYBRID || "";
const verbose = process.env.MEMEX_VERBOSE || "";
const child = spawn(
  "docker",
  [
    "run",
    "-i",
    "--rm",
    "-v",
    "memex-data:/data",
    "-e",
    "MEMEX_DIR=/data",
    "-e",
    `MEMEX_USER_ID=${userID}`,
    "-e",
    `MEMEX_HYBRID=${hybrid}`,
    "-e",
    `MEMEX_VERBOSE=${verbose}`,
    "ghcr.io/kioie/memex:0.6.0",
    "serve",
  ],
  { stdio: "inherit" },
);

child.on("exit", (code, signal) => {
  if (signal) {
    process.kill(process.pid, signal);
    return;
  }
  process.exit(code ?? 1);
});

child.on("error", (err) => {
  console.error(err.message);
  process.exit(1);
});
