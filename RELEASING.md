# Releasing Nummion

Nummion uses the same distribution model as [HEY CLI](https://github.com/basecamp/hey-cli/blob/main/docs/install.md): a tagged GitHub Release contains ready-to-run binaries, package-manager manifests point to those binaries, and installer scripts download the appropriate archive. Every installation exposes `num` (`num.exe` on Windows).

## One-time repository setup

The repository and release configuration use `JangoCG/nummion`. The repository is currently private; public distribution has not been enabled.

1. Review and commit the local changes, then push them to `main` when authorized.
2. Make the GitHub repository public when ready to publish its contents and history. Public installer links and the public Homebrew/Scoop tap require publicly downloadable releases. The workflow refuses publication from a different or private repository.
3. Create a dedicated private GitHub App owned by `JangoCG`, with **Contents: read and write** and the required read-only Metadata permission. Disable webhooks and user authorization; install it only on `JangoCG/homebrew-tap`. In Nummion's **release environment**, store its client ID as the variable `RELEASE_CLIENT_ID` and its private key as the secret `RELEASE_APP_PRIVATE_KEY`. The workflow creates and revokes a short-lived installation token for each stable release. Both Homebrew (`Casks/nummion.rb`) and Scoop (`bucket/nummion.json`) use that token. GitHub's automatic `GITHUB_TOKEN` cannot write to a different repository. Never commit the private key or pass it in a command-line argument.
4. The `release` environment is configured to accept only `v*` tags. Keep GitHub Actions enabled. The release job declares `contents: write` for release uploads and `id-token: write` for keyless signing; CI has read-only permissions.
5. For macOS signing, supply the Apple credentials described below. Review the first generated Homebrew cask and Scoop manifest before promoting the first stable release.

No release, tag, remote rename, visibility change, or secret is created by `make check`, `make release-check`, or `make snapshot`.

## Local checks

```bash
mise install
mise exec -- make check VERSION=0.1.0-dev
mise exec -- go test -race ./...
mise exec -- make snapshot
```

`make snapshot` validates the config and Unix installer tests, builds six binaries (macOS/Linux/Windows × amd64/arm64), generates completions, archives, Linux packages, Homebrew/Scoop manifests, and checksums under `dist/`. It skips signing and publishing. Snapshot builds work with an uncommitted worktree and do not require GitHub tokens.

CI runs Go checks and the race detector on Linux, macOS, and Windows, exercises the installers offline, and builds the full snapshot. The Security workflow adds Gitleaks, govulncheck, gosec, Trivy and GitHub Actions audits. Windows installer integration tests use the freshly built Windows executable. Local macOS validation cannot replace a Windows runner or actual hosted signing/publication.

## Publish

After the final commit is merged to `main` and CI plus Security are green, create and push an annotated semantic-version tag:

```bash
git tag -a v0.1.0 -m 'Nummion v0.1.0'
git push origin v0.1.0
```

Only execute these commands when intentionally publishing. The tag triggers the release workflow. It reruns CI and the Security workflow against the tag commit, verifies that commit belongs to `main`, then GoReleaser builds the artifacts, signs `checksums.txt` with keyless Sigstore, publishes the GitHub Release, and updates the tap. Stable releases require the release GitHub App credentials. A tag such as `v0.1.0-rc.1` creates a prerelease and skips tap updates; it does not replace GitHub's latest stable release.

Assets include:

- `nummion_<version>_<os>_<arch>.tar.gz` (macOS/Linux), or `.zip` (Windows), containing `num`, license notices, README, and shell completions.
- Linux `.deb`, `.rpm`, and `.apk` packages.
- `checksums.txt` and its Sigstore bundle `checksums.txt.bundle`.
- `install.sh` and `install.ps1`, also covered by the release checksums.

If publication fails, inspect the release assets and both tap files before retrying. GitHub Releases and commits to a second repository are not one atomic operation. Do not move or delete a published version tag to repair a failure; prefer a corrected patch release. A failure to update the tap can leave an otherwise usable GitHub Release.

## Signing

Sigstore signs release checksums through GitHub's OIDC identity. It needs no stored signing key. To verify a downloaded release with cosign v3+:

```bash
cosign verify-blob --bundle checksums.txt.bundle \
  --certificate-identity 'https://github.com/JangoCG/nummion/.github/workflows/release.yml@refs/tags/v0.1.0' \
  --certificate-oidc-issuer https://token.actions.githubusercontent.com checksums.txt
sha256sum --check --ignore-missing checksums.txt
```

Replace the example version with the exact release tag. Download the bundle, checksum file, and desired archive into the same directory. On macOS, use `shasum -a 256 <archive>` and compare its output with the matching entry. Both installers always check SHA-256; when cosign is available, signature verification must also succeed. They validate the downloaded executable's version before replacing an existing installation.

Sigstore proves the release's workflow identity; it does not replace operating-system code signing. Optional macOS signing/notarization is configured through these secrets in the **release environment**:

| Secret | Value |
| --- | --- |
| `MACOS_SIGN_P12` | Base64-encoded Developer ID Application certificate with private key (.p12) |
| `MACOS_SIGN_PASSWORD` | Password of that .p12 |
| `MACOS_NOTARY_KEY` | Base64-encoded App Store Connect API private key (.p8) |
| `MACOS_NOTARY_KEY_ID` | API key ID |
| `MACOS_NOTARY_ISSUER_ID` | API issuer ID |

All five must be set to enable notarization. Without them, macOS artifacts are not Developer-ID signed/notarized and Gatekeeper can block downloaded binaries, including Homebrew casks. Configure these before advertising frictionless macOS cask installation. Windows Authenticode signing is not configured; Smart App Control can block unsigned executables. Test distribution on a clean machine before the first stable release. No installer disables OS security controls.

## Installation and updates

After the first stable public release:

```bash
mise use -g github:JangoCG/nummion
# or
brew install --cask JangoCG/tap/nummion
# or
curl -fsSL https://github.com/JangoCG/nummion/releases/latest/download/install.sh | bash

num --version
num auth set
num skill install
```

See the README for PowerShell, Scoop, Go, and package downloads. Updating uses the original package manager or reruns the installer. Package managers handle PATH and completion installation where supported; the standalone installers print the directory to add to PATH and do not modify shell configuration. There is no built-in self-update command yet.

The legacy keychain service, `LEXWARE_API_KEY`, the `lexware` skill name, and the ownership marker remain unchanged so credentials and managed skill installations are reused. `make install` also installs a local `lexware -> num` compatibility symlink. GitHub Release packages install only `num`.

## Relationship to the reference release setup

The security checks and environment-scoped GitHub App token follow [HEY CLI](https://github.com/basecamp/hey-cli/blob/main/RELEASING.md). Nummion scans test files too and pins a newer Gitleaks release. Basecamp organization bots, their 1Password automation, AUR/Nix publishing and their paid Apple/DigiCert identities cannot be copied as usable account configuration. Nummion does not need the Lexware API key in CI or releases.

The GitHub App registration/installation and real signing credentials must be provisioned for the owner's account. Configuration files do not create or contain those credentials. Apple signing remains optional and Windows Authenticode signing is not yet integrated; an unsigned release is not equivalent to HEY's signed distribution.
