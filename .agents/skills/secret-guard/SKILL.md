---
name: secret-guard
description: >-
  Audits, verifies, and protects the codebase against committing sensitive data,
  including .env files, private keys, cloud credentials, tokens, and database passwords.
---

# Secret Guard Skill

This skill provides procedures for auditing the workspace for secrets, testing pre-commit defenses, and remediating accidental secret exposures.

## 1. Quick Audit of Staged Files
To scan currently staged files before committing:
```bash
python ./scripts/secret_scanner.py
```

## 2. Scanning Untracked and Modified Files
To check working directory status for un-ignored `.env` or sensitive files:
```bash
git status --ignored
```

## 3. Remediating Staged Secrets
If a secret or forbidden file was accidentally staged:
```bash
# Unstage the specific file
git restore --staged <file_path>

# Verify .gitignore contains the file pattern
echo "<pattern>" >> .gitignore
```

## 4. Secret Sanitization Guidelines
- **AWS Credentials**: Never commit `AKIA...` keys or `aws_secret_access_key`.
- **Private Keys**: Ensure `*.pem`, `*.key`, `id_rsa` remain in `.gitignore`.
- **JWT & Bearer Tokens**: Do not hardcode test JWT tokens in persistent files.
- **Connection Strings**: Use connection strings parameterized by environment variables (e.g., `postgresql://${DB_USER}:${DB_PASS}@${DB_HOST}:${DB_PORT}/${DB_NAME}`).
- **Templates**: Only commit `.env.example` templates with generic placeholders.
