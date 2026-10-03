import type { Plugin } from "@opencode-ai/plugin";
import { execFileSync } from "node:child_process";

const COMMIT_RE = /\bgit\s+commit\b/i;

export const SecretGuardPlugin: Plugin = async ({ directory }) => {
  return {
    "tool.execute.before": async (input, output) => {
      if (input.tool !== "bash") return;
      const args = output.args as { command?: unknown } | undefined;
      const command = typeof args?.command === "string" ? args.command : "";
      if (!COMMIT_RE.test(command)) return;

      try {
        execFileSync("python", ["./scripts/secret_scanner.py"], { cwd: directory, stdio: "pipe" });
      } catch (error) {
        const e = error as { stdout?: Buffer; stderr?: Buffer; message?: string };
        const detail = `${e.stdout?.toString() ?? ""}${e.stderr?.toString() ?? ""}`.trim();
        throw new Error(
          `Secret-Guard blocked the commit. Remove credentials/.env files from staging before committing.\n${detail || e.message || ""}`,
        );
      }
    },
  };
};
