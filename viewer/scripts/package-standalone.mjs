// File: package-standalone.mjs
// Date: 2026-09-17
// Version/Status: 1.0 / Slice 5 production packaging
// Product/Component: JEH-HOHO / Capital Reservoir Viewer
// Purpose: Add client assets omitted by Next.js standalone tracing to the package.
// Inputs/Outputs: .next/static and optional public in; complete .next/standalone out.
// Invariants: Generated server files remain unchanged; copied assets replace stale copies.
// Failure behavior: Missing required static output fails the production build.
// Non-responsibilities: Source compilation, deployment, or runtime configuration.

import { access, cp, mkdir, rm } from "node:fs/promises";
import { constants } from "node:fs";
import { join } from "node:path";

const root = process.cwd();
const standalone = join(root, ".next", "standalone");
const staticSource = join(root, ".next", "static");
const staticTarget = join(standalone, ".next", "static");

await access(staticSource, constants.R_OK);
await rm(staticTarget, { force: true, recursive: true });
await mkdir(join(standalone, ".next"), { recursive: true });
await cp(staticSource, staticTarget, { recursive: true });

const publicSource = join(root, "public");
const publicTarget = join(standalone, "public");
try {
  await access(publicSource, constants.R_OK);
  await rm(publicTarget, { force: true, recursive: true });
  await cp(publicSource, publicTarget, { recursive: true });
} catch (error) {
  if (error?.code !== "ENOENT") throw error;
}