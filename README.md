# Nummion

A lightweight, unofficial CLI for [Lexware Office](https://developers.lexware.io/docs/). Manage contacts, invoices, vouchers, and documents from your terminal with `num`.

*Nummion is the project name. `num` is the command.*

> [!IMPORTANT]
> This is an unofficial project and is not affiliated with Lexware or Haufe Group.
> `Lexware` and `Lexware Office` are trademarks of their respective owners.

## Features

- Retrieve the account profile and available business functions
- List posting categories
- List, filter, retrieve, create, and update contacts
- List and retrieve invoices, create them as drafts or finalized invoices, and download them
- List, retrieve, upload, and download accounting vouchers, and attach additional files
- Human-readable terminal tables and unmodified API JSON for scripts
- Pagination with `--page`, `--size`, `--all`, and convenient full-year queries
- Secure API key storage in the system keychain or through `LEXWARE_API_KEY`
- `--dry-run` support for every data-writing operation
- Local rate limiting and controlled retries for temporary failures
- Safe downloads without accidental overwrites
- An embedded agent skill for Codex, Claude Code, and the shared agent-skill standard

## Name

Nummion is inspired by the [Byzantine *nummion*](https://tesauros.cultura.gob.es/tesauros/numismatica/1187059.html), a copper coin and later a unit of account. The name connects the project to the history of money and bookkeeping. In the terminal, three letters are enough: `num`.

## Installation

Install a prebuilt release using one of the methods below, or build from source at the end of this section.

### mise (macOS, Linux, Windows)

Nummion distributes prebuilt binaries through GitHub Releases:

```bash
mise use -g github:JangoCG/nummion
num --version
```

### Homebrew (macOS / Linux)

```bash
brew install --cask JangoCG/tap/nummion
num --version
```

### Installer (macOS / Linux / WSL)

```bash
curl -fsSL https://github.com/JangoCG/nummion/releases/latest/download/install.sh | bash
```

The installer downloads the matching binary, verifies its SHA-256 checksum and version, and installs `num` into `~/.local/bin`. If `cosign` v3+ is installed, it also verifies the release signature. It needs no Go toolchain or administrator privileges. If necessary, add the install directory to your shell configuration:

```bash
export PATH="$HOME/.local/bin:$PATH"
```

Set `NUMMION_VERSION` to select a release (including prereleases), and `NUMMION_BIN_DIR` to change the destination. For example:

```bash
curl -fsSL https://github.com/JangoCG/nummion/releases/download/v0.1.0/install.sh | NUMMION_VERSION=0.1.0 bash
```

### Windows (PowerShell)

```powershell
irm https://github.com/JangoCG/nummion/releases/latest/download/install.ps1 | iex
```

This installer uses the same checksum and optional signature verification. It installs `num.exe` into `%LOCALAPPDATA%\Nummion\bin`. Add that directory to your user `Path` environment variable if needed. The same `NUMMION_VERSION` and `NUMMION_BIN_DIR` environment variables are supported.

Or install with Scoop, which handles `PATH` for you:

```powershell
scoop bucket add jangocg https://github.com/JangoCG/homebrew-tap
scoop install jangocg/nummion
```

### Other options

Download archives or Linux `.deb`, `.rpm`, and `.apk` packages from [Releases](https://github.com/JangoCG/nummion/releases). Releases support macOS, Linux, and Windows on amd64 and arm64. Archives include shell completions and `checksums.txt` covers the release artifacts.

With Go installed:

```bash
go install github.com/JangoCG/nummion/cmd/num@latest
```

### From source

[mise](https://mise.jdx.dev/) provides the Go and GoReleaser versions configured by this repository.

```bash
git clone https://github.com/JangoCG/nummion.git
cd nummion
mise install
mise exec -- make install VERSION=0.1.0-dev
num --help
```

By default, `make install` writes `~/.local/bin/num` and a compatibility symlink named `lexware` for existing scripts. Published packages expose `num`. Set `PREFIX=/usr/local` for a different prefix, or run `mise exec -- make build` and use `./bin/num` directly.

Go 1.25 or newer is required; the pinned development version is in [`.mise.toml`](.mise.toml). The existing keychain service (`lexware-cli`) and environment variable (`LEXWARE_API_KEY`) remain compatible with earlier installations.

### Upgrading

Use the method you installed with:

```bash
mise upgrade github:JangoCG/nummion
brew upgrade --cask JangoCG/tap/nummion
```

For Scoop, run `scoop update nummion`. For a script installation, run the installer again. For Go, rerun `go install`. Nummion does not yet have a built-in `num upgrade` command.

Maintainers: see [RELEASING.md](RELEASING.md) for publishing and signature verification.

## Getting started

1. Create a private API key under [Public API](https://app.lexware.de/addons/public-api) in your Lexware Office account.
2. Store the key interactively and securely in the system keychain.
3. Verify the connection and retrieve your first data.

```bash
num auth set
num auth status
num profile
num contacts list
```

`num auth set` reads the key from the terminal without displaying the input. Never place the key in a chat, command-line argument, repository file, or shell history.

## Help and reference

Every command includes built-in help with its usage, flags, and subcommands:

```bash
num --help
num contacts --help
num contacts update --help
num invoices create --help
num vouchers list --help
```

The primary command groups are:

| Area | Command | Purpose |
| --- | --- | --- |
| Authentication | `num auth` | Set, verify, or remove the API key |
| Account | `num profile` | Retrieve the profile and business functions |
| Accounting | `num posting-categories` | Read available posting categories |
| Contacts | `num contacts` | Read, filter, create, and update contacts |
| Invoices | `num invoices` | Read, create, and download invoices |
| Vouchers | `num vouchers` | Manage accounting vouchers and their files |
| Agents | `num skill` | Display or install the embedded agent skill |
| Shell | `num completion` | Generate shell completion scripts |

## Authentication

### System keychain

Interactive setup is the recommended method:

```bash
num auth set
```

The key is stored under the `lexware-cli` service in the system keychain. The CLI never prints it back out.

```bash
num auth status   # verify the key and display the connected account
num auth logout   # remove the stored key
```

`auth status` retrieves the Lexware profile, verifying both that a key is available and that the API accepts it.

### Environment variable

For CI and short-lived shells, an API key that has already been provided securely can be passed through `LEXWARE_API_KEY`. The environment variable takes precedence over the system keychain.

```bash
LEXWARE_API_KEY="…" num profile --json
```

Configure the key through the secret management facility of your CI platform. Do not write it into shell scripts or files in the repository.

The `auth set --token` flag exists for exceptional cases but is unsafe because the value may become visible in shell history and process listings. Use `num auth set` without additional arguments instead.

## Output and global flags

Global flags may appear before or after subcommands:

```bash
num --json contacts list --all
num contacts list --all --json
num --timeout 60s vouchers list --year 2025 --json
```

| Flag | Behavior |
| --- | --- |
| `--json` | Emit compact, machine-readable JSON |
| `--quiet`, `-q` | Suppress human rendering; downloads print only the destination path |
| `--timeout DURATION` | Set the HTTP timeout; defaults to `30s` |
| `--base-url URL` | Override the API base URL; intended for development and local tests |
| `--version`, `-v` | Display the installed CLI version |
| `--help`, `-h` | Display help for the current command |

Without `--json`, detail responses are pretty-printed and listings appear as tables. With `--json`, individual Lexware API responses remain unchanged. When `--all` is used, the CLI combines multiple API pages into one API-compatible page object.

The production base URL defaults to `https://api.lexware.io`. An alternative URL must use HTTPS; unencrypted HTTP is allowed only for loopback addresses in local tests. URLs containing embedded credentials are rejected.

## CLI commands

### Profile and posting categories

```bash
num profile
num profile --json
num posting-categories list
num posting-categories list --json
```

The profile contains the organization and its available business functions. Posting categories help interpret `categoryId` values in bookkeeping voucher items; the API response includes fields such as `id`, `name`, `type`, and `groupName`.

### Contacts

#### Listing and retrieving contacts

```bash
num contacts list
num contacts list --all
num contacts list --name "Sample"
num contacts list --email "office@example.com"
num contacts list --customer
num contacts list --vendor
num contacts list --number 10001
num contacts get CONTACT_ID --json
```

Listings are paginated. A page contains 25 entries by default and at most 250. `--all` starts at page 0, raises smaller page sizes to 250, and retrieves every page. Alternatively, use `--page` and `--size` explicitly.

Lexware requires at least three characters for `--name` and `--email`. The `--customer` and `--vendor` flags filter by the corresponding contact role.

#### Creating a contact

Contacts are read as a JSON object from a file or stdin. If the `version` field is absent during creation, the CLI adds `"version": 0`.

```bash
# Inspect the complete request without sending it
num contacts create --from examples/contact.json --dry-run --json

# Create the contact only after a successful preview
num contacts create --from examples/contact.json --json

# Read JSON from stdin
generate-contact | num contacts create --from - --dry-run --json
```

A template is available at [`examples/contact.json`](examples/contact.json).

#### Updating a contact

Updates are safe partial updates by default: the CLI retrieves the current contact, preserves omitted fields, and sends the current Lexware `version` for optimistic locking.

```bash
num contacts update CONTACT_ID --from contact-update.json --dry-run --json
num contacts update CONTACT_ID --from contact-update.json --json
```

If the current version is already known, it can be supplied explicitly:

```bash
num contacts update CONTACT_ID --from contact-update.json --version 7 --dry-run --json
```

`--replace` sends the supplied JSON as the complete writable contact representation. Use this mode only when a full replacement is intentional:

```bash
num contacts update CONTACT_ID --from complete-contact.json --replace --dry-run --json
num contacts update CONTACT_ID --from complete-contact.json --replace --json
```

### Invoices

#### Listing invoices

```bash
num invoices list
num invoices list --status open --all
num invoices list --contact-id CONTACT_ID --all
num invoices list --number INVOICE_NUMBER
num invoices list --year 2025 --json
```

`invoices list` uses the Lexware voucher list and therefore returns a summary. Use the returned `id` with `invoices get` to retrieve the complete invoice:

```bash
num invoices get INVOICE_ID --json
```

Listings can be filtered by voucher type, status, number, contact, archived state, voucher date, creation date, and update date. Pass multiple types or statuses as comma-separated values.

`--year YYYY` automatically sets the voucher date range from January 1 through December 31, starts at page 0, and retrieves every page. It cannot be combined with `--voucher-date-from` or `--voucher-date-to`.

#### Creating an invoice

An invoice is created as a **draft** by default. Before every real write, run the command with the exact same arguments plus `--dry-run --json`.

```bash
# Preview and create a draft
num invoices create --from examples/invoice.json --dry-run --json
num invoices create --from examples/invoice.json --json

# Finalize immediately only when explicitly intended
num invoices create --from examples/invoice.json --finalize --dry-run --json
num invoices create --from examples/invoice.json --finalize --json

# Read JSON from stdin
generate-invoice | num invoices create --from - --dry-run --json
```

A template is available at [`examples/invoice.json`](examples/invoice.json). The dry run shows the HTTP method, API path, `finalize` state, and normalized request body without sending a request.

> [!CAUTION]
> Do not blindly repeat a failed write after an ambiguous network or server error. Lexware may already have processed it. Inspect the invoice listings and account before making another write attempt.

#### Downloading an invoice

```bash
num invoices download INVOICE_ID --format pdf --output invoice.pdf
num invoices download INVOICE_ID --format xml --output invoice.xml --json
```

Supported formats are `auto`, `pdf`, and `xml`. Without `--output`, the CLI uses the filename suggested by Lexware or a safe fallback. Files are created with owner-only permissions (`0600`) and are never silently overwritten. The `--force` flag must be supplied explicitly to replace a specific destination file.

### Accounting vouchers

#### Listing and retrieving vouchers

```bash
num vouchers list
num vouchers list --type purchaseinvoice --status any
num vouchers list --contact-id CONTACT_ID --all
num vouchers list --year 2025 --json
num vouchers get VOUCHER_ID --json
```

`vouchers list` supports the same filters and pagination as `invoices list`, but its default type is `any`. The listing is also a summary; `vouchers get` returns the detail representation.

The `id` from the voucher list identifies the voucher. Its detail representation contains a `files` array with the file IDs expected by `vouchers download`.

#### Uploading a new voucher

```bash
num vouchers upload purchase-invoice.pdf --dry-run --json
num vouchers upload purchase-invoice.pdf --json
```

Supported formats are PDF, JPG, JPEG, PNG, and XML, with a maximum size of 5 MiB. The dry run locally checks the file's existence, type, and size, then displays the intended API path without uploading any data.

#### Attaching a file to a voucher

```bash
num vouchers attach VOUCHER_ID attachment.pdf --dry-run --json
num vouchers attach VOUCHER_ID attachment.pdf --json
```

`attach` expects the voucher ID, not the ID of an existing file. Uploads and attachments have the same format and size restrictions.

#### Downloading a voucher file

```bash
num vouchers download FILE_ID --output voucher.pdf
num vouchers download FILE_ID --format xml --output voucher.xml --json
```

The file ID comes from the `files` array returned by `vouchers get`. Downloads use the same protections as invoice downloads: mode `0600`, no overwrite without `--force`, and removal of an incomplete destination file after a transfer failure.

## Pagination and full-year queries

`contacts list`, `invoices list`, and `vouchers list` support:

| Flag | Meaning |
| --- | --- |
| `--page N` | Retrieve one page, starting at 0 |
| `--size N` | Set a page size between 1 and 250 |
| `--all` | Combine every page, starting at page 0 |
| `--year YYYY` | Retrieve a complete calendar year for invoices or vouchers; enables `--all` |

Examples:

```bash
num contacts list --page 2 --size 100 --json
num contacts list --all --json
num invoices list --year 2025 --json > invoices-2025.json
num vouchers list --year 2025 --json > vouchers-2025.json
```

A full-year query may require multiple API requests and can therefore take longer. The CLI maintains a local interval of at least 550 milliseconds between requests.

## Automation and scripting

Use `--json` for machine processing:

```bash
num contacts list --all --json | jq '.content[] | {id, company: .company.name}'
num invoices get INVOICE_ID --json > invoice.json
num vouchers list --year 2025 --json > vouchers.json
```

Individual API responses are not wrapped in an additional CLI envelope, so scripts can work directly with Lexware API fields. Local results such as dry runs, installation status, and download paths use a small CLI-owned JSON object.

JSON-writing commands accept `-` for stdin:

```bash
generate-contact | num contacts create --from - --dry-run --json
generate-invoice | num invoices create --from - --dry-run --json
```

The input must contain exactly one JSON object. Multiple values, arrays, and empty inputs are rejected.

### Exit codes

| Code | Meaning |
| ---: | --- |
| 0 | Success |
| 1 | General error |
| 2 | Resource not found |
| 3 | Authentication or permission error |
| 4 | Input, validation, or conflict error |
| 5 | Rate limit |
| 6 | Network, timeout, or server error |

## AI agent integration

Nummion includes an embedded agent skill named `nummion` (`$nummion` in Codex, `/nummion` in Claude Code). It teaches agents the CLI's commands, ID relationships, output formats, and safety rules.

```bash
num skill           # print the embedded SKILL.md to stdout
num skill install   # install the skill globally and for detected agents
```

The installer places the shared copy at `~/.agents/skills/nummion/SKILL.md`.

- Claude Code receives a symlink at `~/.claude/skills/nummion`. If symlinks are unavailable, the files are copied safely instead.
- Codex receives a copy at `$CODEX_HOME/skills/nummion/SKILL.md`, or at `~/.codex/skills/nummion/SKILL.md` by default.
- Agents that are not detected are left unchanged.

Every skill directory managed by the CLI carries a `.managed-by-nummion` marker. An existing unmarked directory, foreign file, or foreign symlink is never claimed or overwritten. After a version change, the CLI refreshes only installed, marked copies.

Upgrading from v0.1.0: run `num skill install` again to install the renamed `nummion` skill. After a successful install, unchanged CLI-managed `lexware` skills from v0.1.0 are removed. Customized or foreign skills remain untouched and are reported for manual review. Update references in your agent instructions from `$lexware` or `/lexware` to `$nummion` or `/nummion`.

In particular, the skill instructs agents to:

- Perform reads with `--json`
- Avoid an authentication preflight before every data command
- Never request, display, or pass API keys as arguments
- Preview writes with `--dry-run --json`
- Create invoices as drafts by default
- Use `--finalize` and `--force` only when explicitly requested
- Never blindly retry ambiguously failed writes

## Security

- **Secrets:** API keys live in the system keychain or an environment variable that has already been configured securely. The CLI does not require a key file.
- **Transport:** The production API is accessed exclusively over HTTPS. HTTP exceptions apply only to loopback tests.
- **Write protection:** Contact, invoice, and voucher mutations provide a non-writing dry run.
- **Invoice state:** `invoices create` always creates a draft unless `--finalize` is supplied.
- **Optimistic locking:** Contact updates use the current Lexware `version` field.
- **Files:** Downloads are created with mode `0600`, never overwrite without `--force`, and do not leave an incomplete file after a failure.
- **Uploads:** The CLI validates permitted file extensions, regular files, empty files, and the 5 MiB limit before uploading.
- **Rate limiting:** The CLI limits its local request rate and honors `Retry-After` for HTTP 429 responses.
- **Retries:** GET and HEAD requests may be retried after network failures and HTTP 502/503 responses. Writes are not retried automatically after HTTP 504.
- **Error output:** API error bodies are read with a size limit, while any Lexware trace ID remains visible for diagnostics.

## Shell completion

The CLI generates completion scripts through Cobra:

```bash
# Bash
source <(num completion bash)

# Zsh
source <(num completion zsh)

# Fish
num completion fish | source
```

PowerShell:

```powershell
num completion powershell | Out-String | Invoke-Expression
```

Each completion subcommand also describes permanent installation:

```bash
num completion zsh --help
```

## Troubleshooting

### No API key configured

Set the key interactively in your own terminal:

```bash
num auth set
```

A missing or rejected key, or missing API permissions, results in exit code 3.

### Timeout or temporary server failure

Increase the timeout for large full-year queries:

```bash
num --timeout 60s vouchers list --year 2025 --json
```

A read can be repeated after its underlying problem has been resolved. Before repeating a write, first verify whether Lexware already processed it.

### Destination file already exists

Downloads do not overwrite existing files. Choose a new destination path, or use `--force` only when that exact file should intentionally be replaced.

### Invalid pagination

`--page` cannot be negative, and `--size` must be between 1 and 250. `--year` must contain four digits and cannot be combined with manual voucher-date boundaries.

### Skill directory is not overwritten

If an unmarked `nummion` skill directory already exists, installation stops intentionally. Move or back up the existing directory yourself, then run:

```bash
num skill install
```

The CLI never deletes or claims foreign skill installations automatically.

## Development

### Repository structure

```text
cmd/nummion/          Program entry point (builds num)
cmd/num/              Compatibility entry point for go install
internal/api/         HTTP client, rate limiting, and API errors
internal/cmd/         Cobra commands and workflows
internal/credentials/ System keychain and environment variable
internal/harness/     Agent detection and skill paths
internal/output/      Table and JSON output
internal/payload/     Safe JSON input
skills/nummion/       Embedded agent skill
examples/             Example payloads
```

### Checks and tests

Run the complete local verification workflow after making changes:

```bash
mise exec -- gofmt -w cmd internal skills
mise exec -- make check VERSION=0.1.0-dev
mise exec -- go test -race ./...
mise exec -- make install VERSION=0.1.0-dev
num skill install
```

`make check` verifies formatting, runs `go vet` and the tests, and builds the binary. API tests use `httptest` and must not modify real Lexware data.

Additional Make targets:

```bash
mise exec -- make build   # build bin/num
mise exec -- make test    # run tests
mise exec -- make vet     # run go vet
mise exec -- make clean   # remove bin/ and dist/
mise exec -- make release-check # validate release config and installers
mise exec -- make snapshot      # build all release packages without publishing
```

New API capabilities belong in the existing general CLI architecture: Cobra commands under `internal/cmd`, HTTP access through `internal/api.Client`, structured output through `internal/output`, dry runs for writes, and pagination for paged endpoints.

## API reference

The implementation uses only the documented Lexware Office base URL, `https://api.lexware.io`.

- [Official API documentation](https://developers.lexware.io/docs/)
- [API key management](https://app.lexware.de/addons/public-api)
- [Public API license and terms of use](https://agb.lexware.de/lexware-office/public-api-lizenz--und-nutzungsbedingungen)

## Security

Credential handling and security checks are described in [SECURITY.md](SECURITY.md). Install local commit and push checks with `mise exec -- make hooks`.

## License

This project is licensed under the MIT License. See [`LICENSE`](LICENSE).

Agent integration and release tooling are based on [HEY CLI](https://github.com/basecamp/hey-cli). See [third-party notices](THIRD_PARTY_NOTICES.md).
