# Security

## Credentials

Use `num auth set` to store your Lexware API key in the system keychain, or provide it through `LEXWARE_API_KEY`. Never commit credentials, include them in issues or logs, or pass them as command-line arguments. Tests and security checks do not need a real Lexware API key.

Release credentials belong in the GitHub `release` environment. Pull-request jobs must not receive them or run untrusted code with privileged credentials.

## Security checks

Local hooks and GitHub Actions scan for secrets, vulnerable dependencies, and unsafe code or workflow configuration. Install the hooks and run local checks with:

```bash
mise install
mise exec -- make hooks
mise exec -- make security-check
```

Local hooks can be bypassed; required CI checks provide the server-side gate. Scanners cannot detect every possible secret. `.gitignore` does not protect files already tracked in Git.

See [RELEASING.md](RELEASING.md) for release checks and signature verification.

## Exposed credentials

Do not paste a live token into a public issue or pull request. Revoke or rotate exposed credentials with their provider first. Deleting a file does not remove it from Git history, clones, or existing logs. Coordinate any history rewrite with maintainers; scanners do not revoke credentials or rewrite history automatically.
