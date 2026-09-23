#!/usr/bin/env python3
"""
Secret and Sensitive Data Scanner for Git and Antigravity Lifecycle Hooks.
Prevents accidental commits of credentials, private keys, API tokens, and .env files.
"""

import sys
import os
import re
import json
import subprocess
from typing import List, Tuple, Dict, Any

# Ensure UTF-8 output on Windows consoles
if hasattr(sys.stdout, "reconfigure"):
    sys.stdout.reconfigure(encoding="utf-8", errors="replace")
if hasattr(sys.stderr, "reconfigure"):
    sys.stderr.reconfigure(encoding="utf-8", errors="replace")

# Filename patterns that should NEVER be committed (unless explicitly .example / .template)
FORBIDDEN_FILE_PATTERNS = [
    # .env files except .env.example, .env.template, .env.sample, .env.dist
    (re.compile(r"(^|/|\\)\.env(\.(?!(example|template|sample|dist)$)[a-zA-Z0-9_\-\.]+)?$", re.IGNORECASE), "Environment variable / secrets file"),
    (re.compile(r"\.(pem|key|pkcs12|pfx|p12|kdbx|keystore|jks)$", re.IGNORECASE), "Private Key / Certificate / Keystore"),
    (re.compile(r"(^|/|\\)(id_rsa|id_dsa|id_ecdsa|id_ed25519)(\.pub)?$", re.IGNORECASE), "SSH Private/Public Key"),
    (re.compile(r"(^|/|\\)(credentials|service-account|serviceAccountKey|google-services)\.json$", re.IGNORECASE), "Cloud / Service Credentials JSON"),
    (re.compile(r"(^|/|\\)local\.properties$", re.IGNORECASE), "Android Local Properties (contains SDK/secrets)"),
]

# High-entropy / known token regex patterns for diff content
SECRET_PATTERNS = [
    (re.compile(r"-----BEGIN (?:RSA |EC |OPENSSH |DSA |PGP |ENCRYPTED )?PRIVATE KEY-----"), "Private Key Header"),
    (re.compile(r"\bAKIA[0-9A-Z]{16}\b"), "AWS Access Key ID"),
    (re.compile(r"(?i)(?:aws_secret_access_key|aws_secret_key|secret_access_key)\s*[:=]\s*[\"']?[A-Za-z0-9/+=]{40}[\"']?"), "AWS Secret Access Key"),
    (re.compile(r"\b(?:ghp|gho|ghu|ghs|ghr)_[A-Za-z0-9_]{36,255}\b"), "GitHub Personal/OAuth Token"),
    (re.compile(r"\bgithub_pat_[A-Za-z0-9_]{82}\b"), "GitHub Fine-Grained Token"),
    (re.compile(r"\bAIza[0-9A-Za-z_\-]{30,40}\b"), "Google API Key"),
    (re.compile(r"\bxox[baprs]-[0-9a-zA-Z]{10,48}\b"), "Slack Token"),
    (re.compile(r"\b(?:sk|rk)_(?:live|test)_[0-9a-zA-Z]{24,99}\b"), "Stripe API Secret/Restricted Key"),
    (re.compile(r"\bSG\.[a-zA-Z0-9_\-]{22}\.[a-zA-Z0-9_\-]{43}\b"), "SendGrid API Key"),
    (re.compile(r"(?:postgres|postgresql|mysql|mongodb(?:\+srv)?|redis):\/\/[^:\s\/]+:[^@\s\/]+@[^\s\/]+"), "Database Connection String with Credentials"),
    (re.compile(r"\beyJ[A-Za-z0-9-_=]{10,}\.[A-Za-z0-9-_=]{10,}\.[A-Za-z0-9-_.+/=]{10,}\b"), "JWT / Bearer Token"),
    (re.compile(r"(?i)(?:api[_-]?key|secret[_-]?key|auth[_-]?token|access[_-]?token|client[_-]?secret|db[_-]?pass(?:word)?)\s*[:=]\s*[\"']([a-zA-Z0-9_\-\.\$\/\+\=]{16,})[\"']"), "Hardcoded Secret / Token Assignment"),
]

# Patterns that indicate false positives / placeholders / test values
SAFE_VALUE_PATTERNS = [
    re.compile(r"(?i)(?:your[_-][a-z0-9_]+|dummy|placeholder|changeme|replace_me|xxxx|<[a-z_-]+>)"),
    re.compile(r"(?i)(?:\$\{[a-z0-9_]+\}|os\.Getenv|process\.env|env\()"),
    re.compile(r"(?i)\b(?:localhost|127\.0\.0\.1|0\.0\.0\.0)\b"),
]


def is_safe_line(line: str) -> bool:
    for safe_pattern in SAFE_VALUE_PATTERNS:
        if safe_pattern.search(line):
            return True
    return False


def scan_staged_files(repo_cwd: str = ".") -> Tuple[List[str], List[str]]:
    """
    Scans staged git files for:
    1. Forbidden file names.
    2. Sensitive patterns in added lines (git diff --cached).
    Returns (violations_filenames, violations_content).
    """
    filename_violations = []
    content_violations = []

    # 1. Check staged filenames
    try:
        res = subprocess.run(
            ["git", "diff", "--cached", "--name-only"],
            cwd=repo_cwd,
            capture_output=True,
            encoding="utf-8",
            errors="replace",
            check=True
        )
        staged_files = [f.strip() for f in res.stdout.splitlines() if f.strip()]
    except Exception:
        # Not a git repo or git error
        return [], []

    for file_path in staged_files:
        for pattern, desc in FORBIDDEN_FILE_PATTERNS:
            if pattern.search(file_path):
                filename_violations.append(f"Forbidden file staged: '{file_path}' ({desc})")

    # 2. Check staged content additions
    try:
        diff_res = subprocess.run(
            ["git", "diff", "--cached", "-U0"],
            cwd=repo_cwd,
            capture_output=True,
            encoding="utf-8",
            errors="replace",
            check=True
        )
        diff_output = diff_res.stdout
    except Exception:
        diff_output = ""

    current_file = "unknown"
    for line in diff_output.splitlines():
        if line.startswith("+++ b/"):
            current_file = line[6:].strip()
            continue
        
        # Skip scanner unit test fixtures and internal planning files
        if "scripts/tests/" in current_file.replace("\\", "/") or ".kilo/" in current_file.replace("\\", "/"):
            continue

        # Only inspect added lines
        if not line.startswith("+") or line.startswith("+++"):
            continue

        added_content = line[1:].strip()
        if not added_content or is_safe_line(added_content):
            continue

        for pattern, desc in SECRET_PATTERNS:
            match = pattern.search(added_content)
            if match:
                # Double check matched value specifically for placeholders
                matched_val = match.group(0)
                if is_safe_line(matched_val):
                    continue
                # Mask secret preview for security
                masked_val = matched_val[:4] + "*" * (len(matched_val) - 8) + matched_val[-4:] if len(matched_val) > 8 else "***"
                content_violations.append(
                    f"Sensitive pattern detected in '{current_file}': {desc} [matched: {masked_val}]"
                )

    return filename_violations, content_violations


def run_as_git_hook(repo_cwd: str = ".") -> int:
    """Runs as a standard pre-commit CLI hook."""
    print("🔒 [Secret-Guard] Scanning staged changes for sensitive data and secrets...")
    file_viols, content_viols = scan_staged_files(repo_cwd)
    all_viols = file_viols + content_viols

    if all_viols:
        print("\n❌ COMMIT BLOCKED! Sensitive information or forbidden files detected:\n")
        for v in all_viols:
            print(f"  • {v}")
        print("\n💡 Please remove credentials / .env files from git staging before committing.")
        print("   To unstage a file: git restore --staged <file>\n")
        return 1

    print("✅ [Secret-Guard] No sensitive data or forbidden files detected.")
    return 0


def run_as_lifecycle_hook() -> int:
    """Runs as an Antigravity PreToolUse lifecycle hook (reading stdin JSON)."""
    try:
        input_data = sys.stdin.read()
        if not input_data.strip():
            print(json.dumps({"decision": "allow"}))
            return 0
        
        payload = json.loads(input_data)
        tool_call = payload.get("toolCall", {})
        tool_name = tool_call.get("name", "")
        tool_args = tool_call.get("args", {})

        # If tool is run_command, check if it's committing
        if tool_name == "run_command":
            cmd = tool_args.get("CommandLine", "")
            cwd = tool_args.get("Cwd", ".")
            
            # Check for git commit commands
            if re.search(r"\bgit\s+commit\b", cmd, re.IGNORECASE):
                file_viols, content_viols = scan_staged_files(cwd)
                all_viols = file_viols + content_viols
                if all_viols:
                    reasons = "\n".join([f"- {v}" for v in all_viols])
                    output = {
                        "decision": "deny",
                        "reason": f"Commit blocked by Secret-Guard. Sensitive data detected in staged changes:\n{reasons}\n\nPlease unstage or remove secrets before committing."
                    }
                    print(json.dumps(output))
                    return 0

        # Allow other tool calls by default
        print(json.dumps({"decision": "allow"}))
        return 0
    except Exception as e:
        # Fallback to allow on unexpected hook failure so we don't hard crash the agent unexpectedly
        print(json.dumps({"decision": "allow", "reason": f"Hook error: {str(e)}"}))
        return 0


def main():
    if "--hook" in sys.argv:
        sys.exit(run_as_lifecycle_hook())
    else:
        sys.exit(run_as_git_hook())


if __name__ == "__main__":
    main()
