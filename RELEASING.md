# Releasing Nummion

Version tags trigger [the release workflow](.github/workflows/release.yml). It runs CI and security checks, builds packages with GoReleaser, signs checksums with Sigstore, and publishes a GitHub Release. Stable releases also update the Homebrew cask and Scoop manifest.

For installation and updates, see the [README](README.md#installation).

## Before publishing

Merge the release commit into `main` with all required checks passing. Validate locally:

```bash
mise install
mise exec -- make check VERSION=0.1.0-dev
mise exec -- go test -race ./...
mise exec -- make snapshot
```

`make snapshot` builds packages under `dist/` without signing or publishing. It does not require release credentials.

## Publish a version

Create and push an annotated tag for the intended version. For example, for a new `v0.1.1` release:

```bash
git tag -a v0.1.1 -m 'Nummion v0.1.1'
git push origin v0.1.1
```

The workflow requires a public `JangoCG/nummion` repository and a tag pointing to a commit on `main`. A tag such as `v0.1.1-rc.1` publishes a prerelease without updating the tap or replacing the latest stable release.

After publication, check the release assets and both package-manager manifests. If a job fails, inspect what was published before retrying. Do not move or delete a published version tag; use a corrected patch release.

## Release credentials

The `release` environment accepts `v*` tags. Stable releases require the `RELEASE_CLIENT_ID` variable and `RELEASE_APP_PRIVATE_KEY` secret for a GitHub App with Contents write access only to `JangoCG/homebrew-tap`. The workflow creates a short-lived installation token and revokes it at completion. Never commit the private key.

## Verify a download

Download `checksums.txt`, `checksums.txt.bundle`, and the desired archive from the same release. With cosign v3+, replace the example tag with that release's version:

```bash
cosign verify-blob --bundle checksums.txt.bundle \
  --certificate-identity 'https://github.com/JangoCG/nummion/.github/workflows/release.yml@refs/tags/v0.1.1' \
  --certificate-oidc-issuer https://token.actions.githubusercontent.com checksums.txt
sha256sum --check --ignore-missing checksums.txt
```

On macOS, use `shasum -a 256 <archive>` and compare it with the matching checksum. Both installers verify SHA-256 and also require signature verification to succeed when cosign is available.

## Platform signing

Sigstore verifies the release workflow's identity; it does not replace operating-system code signing. Optional macOS signing and notarization require all five secrets in the `release` environment:

| Secret | Value |
| --- | --- |
| `MACOS_SIGN_P12` | Base64-encoded Developer ID Application certificate and private key (.p12) |
| `MACOS_SIGN_PASSWORD` | Password for the .p12 |
| `MACOS_NOTARY_KEY` | Base64-encoded App Store Connect API private key (.p8) |
| `MACOS_NOTARY_KEY_ID` | API key ID |
| `MACOS_NOTARY_ISSUER_ID` | API issuer ID |

Without these, macOS binaries are not Developer-ID signed or notarized. Windows Authenticode signing is not configured. Operating-system security controls may block unsigned downloads; test installation on clean machines. Installers do not disable those controls.
