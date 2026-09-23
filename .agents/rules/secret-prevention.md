---
trigger: always_on
---

# Sensitive Data & Secret Prevention

## Mandate
Under no circumstances should sensitive credentials, secret tokens, private keys, or environment configuration files containing production/local secrets be committed to this repository.

## Rules & Best Practices

1. **Zero Hardcoded Credentials**:
   - Never embed real API keys, bearer tokens, AWS/GCP credentials, database passwords, or private keys directly in code or tests.
   - Use environment variables (e.g., `os.Getenv(...)`, `process.env.*`) or external configuration providers.

2. **Environment Files (`.env`)**:
   - `.env` files with actual configuration values must remain strictly ignored by Git.
   - Commit only `.env.example` templates containing safe placeholder values (e.g., `DB_PASSWORD=your_password_here`).

3. **Pre-Commit Secret Verification**:
   - Staged files and diffs are automatically scanned via `scripts/secret_scanner.py` prior to commit.
   - If a secret or forbidden file is detected, immediately unstage it with `git restore --staged <file>` and eliminate the sensitive value before proceeding.
