import type { Plugin } from "@opencode-ai/plugin";
import { execFile } from "node:child_process";
import { createHash } from "node:crypto";
import { existsSync, mkdirSync, readFileSync, readdirSync, statSync, writeFileSync } from "node:fs";
import { join, relative, sep } from "node:path";

const MANIFEST_REL = join(".agents", "state", "last-tested.json");
const STATE_DIR_REL = join(".agents", "state");
const SKIP_DIRS = new Set(["vendor", "node_modules", ".git", ".kilo", ".opencode", ".agents"]);

type Manifest = {
  version: 1;
  checkedAt: string;
  files: Record<string, string>;
};

function toPosix(path: string): string {
  return path.split(sep).join("/");
}

function listGoFiles(root: string): string[] {
  const out: string[] = [];
  const walk = (dir: string): void => {
    let entries: string[];
    try {
      entries = readdirSync(dir);
    } catch {
      return;
    }
    for (const name of entries) {
      const full = join(dir, name);
      let stat;
      try {
        stat = statSync(full);
      } catch {
        continue;
      }
      if (stat.isDirectory()) {
        if (SKIP_DIRS.has(name)) continue;
        walk(full);
      } else if (name.endsWith(".go")) {
        out.push(full);
      }
    }
  };
  walk(root);
  return out;
}

function hashFile(path: string): string | undefined {
  try {
    return createHash("sha256").update(readFileSync(path)).digest("hex");
  } catch {
    return undefined;
  }
}

function runGoTest(dir: string): Promise<{ code: number; output: string }> {
  return new Promise((resolve) => {
    execFile(
      "go",
      ["test", "./..."],
      { cwd: dir, encoding: "utf8", timeout: 300000, windowsHide: true },
      (err, stdout, stderr) => {
        const output = `${stdout ?? ""}${stderr ?? ""}`;
        const code = err ? ((err as { code?: number }).code ?? 1) : 0;
        resolve({ code, output });
      },
    );
  });
}

function isExcluded(rel: string): boolean {
  return rel === "docs/docs.go" || rel.startsWith("cmd/");
}

export const AutoTestsPlugin: Plugin = async ({ directory }) => {
  const manifestPath = join(directory, MANIFEST_REL);
  let running = false;

  return {
    event: async ({ event }) => {
      if (!event || event.type !== "session.idle") return;
      if (running) return;

      running = true;
      try {
        const current: Record<string, string> = {};
        for (const file of listGoFiles(directory)) {
          const rel = toPosix(relative(directory, file));
          const hash = hashFile(file);
          if (hash) current[rel] = hash;
        }

        let manifest: Manifest | undefined;
        if (existsSync(manifestPath)) {
          try {
            manifest = JSON.parse(readFileSync(manifestPath, "utf8")) as Manifest;
          } catch {
            manifest = undefined;
          }
        }

        const writeManifest = (): void => {
          const next: Manifest = { version: 1, checkedAt: new Date().toISOString(), files: current };
          mkdirSync(join(directory, STATE_DIR_REL), { recursive: true });
          writeFileSync(manifestPath, `${JSON.stringify(next, null, 2)}\n`, "utf8");
        };

        // First run only establishes a baseline; it does not test or warn.
        if (!manifest || !manifest.files) {
          writeManifest();
          return;
        }

        const changed = new Set<string>();
        for (const [rel, hash] of Object.entries(current)) {
          if (manifest.files[rel] !== hash) changed.add(rel);
        }
        for (const rel of Object.keys(manifest.files)) {
          if (!(rel in current)) changed.add(rel);
        }
        if (changed.size === 0) return;

        const changedSource = [...changed].filter(
          (rel) => rel.endsWith(".go") && !rel.endsWith("_test.go") && !isExcluded(rel),
        );
        const changedTests = [...changed].filter((rel) => rel.endsWith("_test.go"));

        const { code, output } = await runGoTest(directory);
        const verdict = code === 0 ? "PASS" : "FAIL";
        console.log(
          `[auto-tests] go test ./... -> ${verdict} (changed: ${changed.size}, source: ${changedSource.length}, tests: ${changedTests.length})`,
        );
        if (code !== 0) {
          console.error(`[auto-tests] go test output:\n${output}`);
        }
        if (changedSource.length > 0 && changedTests.length === 0) {
          console.warn(`[auto-tests] source changed without test changes: ${changedSource.join(", ")}`);
        }

        writeManifest();
      } catch (error) {
        console.error("[auto-tests] guardrail failed:", error);
      } finally {
        running = false;
      }
    },
  };
};
