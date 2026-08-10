import { cpSync, mkdtempSync, rmSync } from "node:fs";
import { tmpdir } from "node:os";
import { join } from "node:path";

import { test as base } from "@playwright/test";

import { type CodeServerContext, startCodeServer } from "./utils_code_server";

interface TempDir {
  path: string;
  dispose(): void;
}

// A temp dir with explicit cleanup so the test runner remains compatible with
// Node 22, which code-server requires.
function makeTempDir(prefix: string): TempDir {
  const path = mkdtempSync(join(tmpdir(), prefix));
  return {
    path,
    dispose() {
      rmSync(path, { recursive: true, force: true });
    },
  };
}

// Shared scratch dir populated by extension.setup.ts: the installed extension
// and the freshly built tally LSP binary.
const SETUP_DIR = join(__dirname, "..", ".test_setup");
const EXTENSIONS_DIR = join(SETUP_DIR, "extensions");
const TALLY_BIN =
  process.env.TALLY_BIN ?? join(SETUP_DIR, process.platform === "win32" ? "tally.exe" : "tally");
const FIXTURE_DOCKERFILE = join(__dirname, "fixtures", "Dockerfile");

interface WorkerFixtures {
  sharedCodeServer: CodeServerContext;
}

interface TestFixtures {
  /** An isolated workspace folder seeded with the lint-bait Dockerfile. */
  projectDir: string;
}

export const test = base.extend<TestFixtures, WorkerFixtures>({
  // One code-server per worker (workers:1 → effectively once per run), using a
  // throwaway user-data dir. The extensions dir is shared and read-only here.
  sharedCodeServer: [
    async ({}, use) => {
      const userData = makeTempDir("tally-e2e-udd-");
      try {
        const ctx = await startCodeServer({
          extensionsDir: EXTENSIONS_DIR,
          userDataDir: userData.path,
          tallyBinaryPath: TALLY_BIN,
        });
        try {
          await use(ctx);
        } finally {
          await ctx[Symbol.asyncDispose]();
        }
      } finally {
        userData.dispose();
      }
    },
    { scope: "worker" },
  ],

  // A fresh project folder per test so edits/fixes never bleed across cases.
  projectDir: async ({}, use) => {
    const project = makeTempDir("tally-e2e-proj-");
    try {
      cpSync(FIXTURE_DOCKERFILE, join(project.path, "Dockerfile"));
      await use(project.path);
    } finally {
      project.dispose();
    }
  },
});

export { expect } from "@playwright/test";
