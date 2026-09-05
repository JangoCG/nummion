# Security

Nummion uses the system keychain or `LEXWARE_API_KEY` for Lexware credentials. Keys, signing certificates, passwords and private keys must never be committed, pasted into issues, or passed as command-line arguments. Use `num auth set` interactively for Lexware authentication.

## Checks before changes reach users

- Gitleaks scans staged files before commits and all reachable history before pushes, including test files. Install the local hooks with `mise exec -- make hooks`.
- GitHub Actions runs the full-history secret scan on pull requests, main pushes, weekly, and before releases. Output is fully redacted. Synthetic regression fixtures verify that staged test-file secrets and subsequently deleted secrets are detected.
- Govulncheck, gosec, Trivy, actionlint and zizmor check dependencies, Go code and workflows. Reviewed gosec exceptions are documented at individual source lines; entire rule families are not disabled.
- Dependabot checks Go modules and GitHub Actions weekly. GitHub dependency alerts and automatic security-update PRs are enabled.
- CodeQL and dependency review are configured for public repositories. For a private repository with licensed code scanning, set the repository variable `CODEQL_ENABLED=true`; otherwise those jobs are explicitly skipped, not reported as successful scans.
- Releases must pass CI and the Security workflow, use a tag on `main`, and run in the `release` environment. The environment accepts only `v*` tags. A GitHub App creates a short-lived token restricted to the Homebrew tap; the token is revoked at job completion.

GitHub's native secret scanning and push protection are enabled for this public repository. The Gitleaks checks run independently. Local Git hooks can be bypassed, so required pull-request checks provide the server-side gate. Neither pattern matching nor static analysis can guarantee detection of every possible secret format.

## Local setup

```bash
mise install
mise exec -- make hooks
mise exec -- make security-check
```

Secret scanning tools never need your actual Lexware API key. Do not supply it to tests or scanner commands. `.gitignore` excludes common local secret files, but ignore rules are not a security boundary and do not remove files already tracked in Git.

## Release credentials

Store release credentials under **Settings → Environments → release**. The checked-in workflow contains only secret references. See [RELEASING.md](RELEASING.md) for the GitHub App and signing prerequisites. Never grant release secrets to pull-request jobs or execute fork code from a privileged `pull_request_target` workflow.

## Reporting a suspected leak

Do not put a live token into a public issue or pull request. Revoke or rotate an exposed credential with its provider first. Removing it from the latest file does not remove Git history, clones or existing logs. Review any history rewrite separately and coordinate with repository users; the scanner does not rewrite history or revoke credentials automatically.
