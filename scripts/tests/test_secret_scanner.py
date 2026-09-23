#!/usr/bin/env python3
import unittest
import json
import io
import sys
from unittest.mock import patch, MagicMock

# Import the scanner
sys.path.insert(0, ".")
from scripts import secret_scanner


class TestSecretScanner(unittest.TestCase):

    def test_forbidden_filename_patterns(self):
        forbidden_samples = [
            ".env",
            ".env.local",
            ".env.production",
            ".env.staging",
            "server/.env",
            "config/id_rsa",
            "keys/server.key",
            "keys/cert.pem",
            "auth/credentials.json",
            "android/local.properties",
            "config/serviceAccountKey.json",
        ]
        for fname in forbidden_samples:
            matched = False
            for pattern, _ in secret_scanner.FORBIDDEN_FILE_PATTERNS:
                if pattern.search(fname):
                    matched = True
                    break
            self.assertTrue(matched, f"Expected {fname} to be flagged as forbidden")

    def test_allowed_filename_patterns(self):
        allowed_samples = [
            ".env.example",
            ".env.template",
            ".env.sample",
            ".env.dist",
            "server/.env.example",
            "src/utils/crypto.go",
            "README.md",
            "package.json",
        ]
        for fname in allowed_samples:
            matched = False
            for pattern, _ in secret_scanner.FORBIDDEN_FILE_PATTERNS:
                if pattern.search(fname):
                    matched = True
                    break
            self.assertFalse(matched, f"Expected {fname} to be allowed, but it was flagged")

    def test_secret_content_patterns(self):
        sensitive_lines = [
            "-----BEGIN " + "RSA PRIVATE KEY-----",
            "AKIA" + "IOSFODNN7EXAMPLE",
            "ghp_" + "1234567890abcdefghijklmnopqrstuvwxyzAB",
            "AIza" + "SyD-1234567890abcdefghijklmnopqr",
            "postgres://" + "admin:SuperSecretPass123!@db.banana.internal:5432/main",
            "mongodb+srv://" + "root:MyP@ssw0rd99@cluster0.abcde.mongodb.net/test",
            "xoxb-" + "123456789012-1234567890123-abcdefghijklmnopqrstuvwx",
            "sk_live_" + "51Abcdefghijklmnopqrstuvwx1234567890",
            'api_key = "' + "a1b2c3d4e5f6g7h8i9j0k1l2m3n4o5p6" + '"',
        ]
        for line in sensitive_lines:
            matched = False
            for pattern, _ in secret_scanner.SECRET_PATTERNS:
                if pattern.search(line):
                    matched = True
                    break
            self.assertTrue(matched, f"Expected content line to be flagged: {line}")

    def test_safe_placeholder_lines(self):
        safe_lines = [
            'API_KEY="your_api_key_here"',
            'DB_PASS="<password>"',
            'SECRET_TOKEN="placeholder"',
            'token = os.Getenv("SECRET_KEY")',
            'const secret = process.env.API_KEY',
            'postgres://user:password@localhost:5432/db',
        ]
        for line in safe_lines:
            self.assertTrue(secret_scanner.is_safe_line(line), f"Expected line to be recognized as safe: {line}")

    def test_lifecycle_hook_blocks_git_commit_when_secrets_found(self):
        mock_payload = {
            "toolCall": {
                "name": "run_command",
                "args": {
                    "CommandLine": "git commit -m 'feat: add config'",
                    "Cwd": "."
                }
            }
        }
        with patch("sys.stdin", io.StringIO(json.dumps(mock_payload))), \
             patch("scripts.secret_scanner.scan_staged_files", return_value=(["Forbidden file staged: .env"], [])):
            captured_stdout = io.StringIO()
            with patch("sys.stdout", captured_stdout):
                secret_scanner.run_as_lifecycle_hook()
            
            output = json.loads(captured_stdout.getvalue())
            self.assertEqual(output.get("decision"), "deny")
            self.assertIn("Forbidden file staged", output.get("reason", ""))

    def test_lifecycle_hook_allows_git_commit_when_clean(self):
        mock_payload = {
            "toolCall": {
                "name": "run_command",
                "args": {
                    "CommandLine": "git commit -m 'feat: add config'",
                    "Cwd": "."
                }
            }
        }
        with patch("sys.stdin", io.StringIO(json.dumps(mock_payload))), \
             patch("scripts.secret_scanner.scan_staged_files", return_value=([], [])):
            captured_stdout = io.StringIO()
            with patch("sys.stdout", captured_stdout):
                secret_scanner.run_as_lifecycle_hook()
            
            output = json.loads(captured_stdout.getvalue())
            self.assertEqual(output.get("decision"), "allow")

    def test_lifecycle_hook_allows_non_commit_tools(self):
        mock_payload = {
            "toolCall": {
                "name": "run_command",
                "args": {
                    "CommandLine": "go test ./...",
                    "Cwd": "."
                }
            }
        }
        with patch("sys.stdin", io.StringIO(json.dumps(mock_payload))):
            captured_stdout = io.StringIO()
            with patch("sys.stdout", captured_stdout):
                secret_scanner.run_as_lifecycle_hook()
            
            output = json.loads(captured_stdout.getvalue())
            self.assertEqual(output.get("decision"), "allow")


if __name__ == "__main__":
    unittest.main()
