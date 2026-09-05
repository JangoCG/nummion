#!/usr/bin/env bash
# Nummion release installer. No Go toolchain or administrator privileges needed.
set -euo pipefail

fail() { printf 'Nummion: %s\n' "$*" >&2; exit 1; }
repo=JangoCG/nummion
fetch() { curl --proto '=https' --tlsv1.2 -fsSL --retry 3 "$@"; }
command -v curl >/dev/null || fail 'curl is required.'
command -v tar >/dev/null || fail 'tar is required.'

case "$(uname -s)" in
  Darwin) platform=darwin ;;
  Linux) platform=linux ;;
  *) fail 'Use install.ps1 on Windows; supported Unix systems are macOS and Linux.' ;;
esac
case "$(uname -m)" in
  x86_64|amd64) arch=amd64 ;;
  arm64|aarch64) arch=arm64 ;;
  *) fail 'Supported architectures are amd64 and arm64.' ;;
esac

version=${NUMMION_VERSION:-}
if [[ -z "$version" ]]; then
  latest=$(fetch -o /dev/null -w '%{url_effective}' "https://github.com/$repo/releases/latest")
  [[ "$latest" == "https://github.com/$repo/releases/tag/"* ]] || fail 'No public release found.'
  version=${latest##*/}
fi
version=${version#v}
[[ "$version" =~ ^(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)(-[0-9A-Za-z]+([.-][0-9A-Za-z]+)*)?$ ]] || fail 'Invalid release version.'
archive="nummion_${version}_${platform}_${arch}.tar.gz"
base="https://github.com/$repo/releases/download/v$version"
work=$(mktemp -d)
staged=''
cleanup() {
  rm -rf "$work"
  if [[ -n "$staged" ]]; then rm -f "$staged"; fi
}
trap cleanup EXIT
fetch -o "$work/$archive" "$base/$archive"
fetch -o "$work/checksums.txt" "$base/checksums.txt"

if command -v cosign >/dev/null; then
  # Use cosign v3+ for the new bundle format. Verification errors are fatal.
  fetch -o "$work/checksums.txt.bundle" "$base/checksums.txt.bundle"
  cosign verify-blob --bundle "$work/checksums.txt.bundle" \
    --certificate-identity "https://github.com/$repo/.github/workflows/release.yml@refs/tags/v$version" \
    --certificate-oidc-issuer https://token.actions.githubusercontent.com "$work/checksums.txt" \
    || fail 'Signature verification failed (cosign v3+ is required).'
fi
expected=$(awk -v file="$archive" '$2 == file {print $1}' "$work/checksums.txt")
[[ "$expected" =~ ^[0-9a-fA-F]{64}$ ]] || fail 'Missing, duplicate, or invalid SHA-256 checksum.'
if command -v sha256sum >/dev/null; then
  actual=$(sha256sum "$work/$archive")
elif command -v shasum >/dev/null; then
  actual=$(shasum -a 256 "$work/$archive")
else
  fail 'sha256sum or shasum is required.'
fi
[[ "${actual%% *}" == "$expected" ]] || fail 'SHA-256 mismatch; existing installation was left untouched.'

tar -xzf "$work/$archive" -C "$work" num
[[ -f "$work/num" && ! -L "$work/num" ]] || fail 'Archive does not contain a regular num executable.'
chmod 0755 "$work/num"
[[ "$("$work/num" --version)" == "num $version" ]] || fail 'Downloaded executable reports an unexpected version.'

bin_dir=${NUMMION_BIN_DIR:-"$HOME/.local/bin"}
mkdir -p "$bin_dir"
[[ ! -d "$bin_dir/num" ]] || fail 'The destination num is a directory.'
staged=$(mktemp "$bin_dir/.num-install.XXXXXX")
install -m 0755 "$work/num" "$staged"
mv -f "$staged" "$bin_dir/num"
staged=''
printf 'Installed num %s to %s/num\n' "$version" "$bin_dir"
case ":$PATH:" in
  *":$bin_dir:"*) ;;
  *) printf 'Add this directory to your shell PATH: %s\n' "$bin_dir" ;;
esac
printf 'Next: num auth set\nOptional agent integration: num skill install\n'
