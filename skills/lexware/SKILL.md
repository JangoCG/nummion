---
name: lexware
description: |
  Interact with Lexware Office through lexware-cli. Use it to inspect the
  account profile, list posting categories, read contacts, invoices and vouchers, create or
  update contacts, create invoice drafts, download invoices and voucher files, and upload or
  attach voucher files. Use for any supported Lexware Office request, including
  questions about the user's contacts, invoices, invoice details, yearly
  invoice or voucher listings, voucher details, and document uploads.
---

# /lexware - Lexware Office CLI

Use `lexware` to work with the Lexware Office Public API from the terminal.

## Agent Invariants

**MUST follow these rules:**

1. **Use structured output** — append `--json` to reads so identifiers and values can be handled without parsing human tables. Lexware returns its API JSON unchanged.
2. **Reuse stored authentication** — run the requested data command directly. It uses `LEXWARE_API_KEY` when present, otherwise the API key stored in the user's system keychain. Use `lexware auth status --json` only when the user asks about authentication or an auth failure needs diagnosis.
3. **Never handle the API key in chat or shell arguments** — never ask the user to paste it into the conversation, never inspect or print `LEXWARE_API_KEY`, and never use `lexware auth set --token`. If authentication is missing, report the task as blocked and tell the user to run `lexware auth set` interactively.
4. **Preview writes** — use `--dry-run --json` first for contact creation or updates, invoice creation, voucher uploads, and voucher attachments. Check the resulting method, path, flags, file, and body before performing the requested write.
5. **Keep invoices as drafts by default** — `lexware invoices create` creates a draft. Add `--finalize` only when the user explicitly asks for a final invoice and the dry-run confirms `"finalize": true`.
6. **Protect local files** — invoice and voucher-file downloads never overwrite by default. Do not add `--force` unless the user explicitly asks to replace that exact destination.
7. **Do not broaden scope** — a year listing already fetches all pages. Do not turn a list request into one API call per invoice unless the user also asks for every invoice's full details.

## Quick Reference

| Task | Command |
|------|---------|
| Show account profile | `lexware profile --json` |
| List posting categories | `lexware posting-categories list --json` |
| Check authentication | `lexware auth status --json` |
| List contacts | `lexware contacts list --json` |
| List every contact | `lexware contacts list --all --json` |
| Filter contacts by name | `lexware contacts list --name "Muster" --json` |
| Read one contact | `lexware contacts get <contact_id> --json` |
| Preview contact creation | `lexware contacts create --from contact.json --dry-run --json` |
| Create a contact | `lexware contacts create --from contact.json --json` |
| Preview contact update | `lexware contacts update <contact_id> --from update.json --dry-run --json` |
| Update a contact | `lexware contacts update <contact_id> --from update.json --json` |
| List invoices | `lexware invoices list --json` |
| List every invoice in a year | `lexware invoices list --year 2025 --json` |
| Read full invoice details | `lexware invoices get <invoice_id> --json` |
| Preview an invoice draft | `lexware invoices create --from invoice.json --dry-run --json` |
| Create an invoice draft | `lexware invoices create --from invoice.json --json` |
| Preview a final invoice | `lexware invoices create --from invoice.json --finalize --dry-run --json` |
| Create a final invoice | `lexware invoices create --from invoice.json --finalize --json` |
| Download an invoice | `lexware invoices download <invoice_id> --format pdf --output invoice.pdf --json` |
| List vouchers | `lexware vouchers list --json` |
| List every voucher in a year | `lexware vouchers list --year 2025 --json` |
| Read voucher details | `lexware vouchers get <voucher_id> --json` |
| Download a voucher file | `lexware vouchers download <file_id> --output voucher.pdf --json` |
| Preview a voucher upload | `lexware vouchers upload receipt.pdf --dry-run --json` |
| Upload a voucher | `lexware vouchers upload receipt.pdf --json` |
| Preview a voucher attachment | `lexware vouchers attach <voucher_id> attachment.pdf --dry-run --json` |
| Attach a file to a voucher | `lexware vouchers attach <voucher_id> attachment.pdf --json` |

## Decision Trees

### Reading invoices and vouchers

```text
Need accounting documents?
├── Invoices for one year? → lexware invoices list --year <YYYY> --json
├── All voucher types for one year? → lexware vouchers list --year <YYYY> --json
├── One invoice's full contents? → lexware invoices get <invoice_id> --json
├── One voucher's details? → lexware vouchers get <voucher_id> --json
├── The invoice file? → lexware invoices download <invoice_id> --format pdf
└── An attached voucher file? → get the voucher, then `lexware vouchers download <file_id>`
```

The `id` returned by an invoice list is used with `invoices get` and `invoices download`. The `id` returned by a voucher list is used with `vouchers get` and `vouchers attach`. A voucher's `files` array contains the file IDs used with `vouchers download`.

### Creating an invoice

```text
Need to create an invoice?
├── Prepare valid Lexware invoice JSON
├── Run create with --dry-run --json
├── User asked for a draft? → create without --finalize
└── User explicitly asked to finalize? → verify dry-run, then create with --finalize
```

Finalization is materially different from creating a draft. Never infer it from the word "create".

## Resource Reference

### Global flags and output

Global flags may appear before or after subcommands:

```bash
lexware --json invoices list --year 2025
lexware invoices list --year 2025 --json
lexware --timeout 60s vouchers list --all --json
```

Prefer `--json`. `--quiet` also suppresses human rendering, but it is intended for scripts where only machine-readable results should be emitted.

### Contacts

```bash
lexware contacts list --all --json
lexware contacts list --name Muster --json
lexware contacts list --email office@example.com --json
lexware contacts list --customer --json
lexware contacts get <contact_id> --json
```

Listings are paginated. Use `--all` to fetch every page, or use `--page` and `--size` explicitly. A name or email filter must contain at least three characters because Lexware applies that constraint.

### Posting categories

```bash
lexware posting-categories list --json
```

The response is the complete unpaginated list from the official posting-categories endpoint. Use its `id`, `name`, `type`, and `groupName` fields to interpret the `categoryId` values in bookkeeping voucher items. Categories with type `outgo` apply to purchase invoices and purchase credit notes; categories with type `income` apply to sales invoices and sales credit notes.

Contact creation and updates read JSON from a file or stdin:

```bash
lexware contacts create --from contact.json --dry-run --json
lexware contacts create --from contact.json --json
generate-contact | lexware contacts create --from - --dry-run --json
lexware contacts update <contact_id> --from update.json --dry-run --json
lexware contacts update <contact_id> --from update.json --json
```

An update is a merge by default. The CLI first reads the current contact, keeps omitted fields, and applies the current `version` for optimistic locking. Use `--replace` only when the user explicitly wants the supplied JSON to replace the full writable contact representation. A contact create adds `version: 0` when it is missing.

### Invoice and voucher listings

```bash
lexware invoices list --year 2025 --json
lexware invoices list --status open --all --json
lexware invoices list --contact-id <contact_id> --all --json
lexware vouchers list --year 2025 --json
lexware vouchers list --type purchaseinvoice --status any --all --json
```

`--year YYYY` sets the voucher date range from January 1 through December 31, starts at page zero, raises the page size to 250, and fetches every page automatically. Do not combine it with `--voucher-date-from` or `--voucher-date-to`.

The list uses Lexware's voucher-list representation, which is a summary. Fetch a single resource for full details:

```bash
lexware invoices get <invoice_id> --json
lexware vouchers get <voucher_id> --json
```

### Invoice creation

Invoice JSON is accepted from a file or stdin:

```bash
lexware invoices create --from invoice.json --dry-run --json
lexware invoices create --from invoice.json --json
generate-invoice | lexware invoices create --from - --dry-run --json
```

The default is a draft. For finalization, use `--finalize` in both the preview and real command. After any ambiguous network or server failure, do not retry a write automatically: Lexware may already have processed it. Read the relevant listings first and report the uncertainty.

### Downloads

```bash
lexware invoices download <invoice_id> --format pdf --output invoice.pdf --json
lexware invoices download <invoice_id> --format xml --output invoice.xml --json
lexware vouchers download <file_id> --format auto --output voucher-file --json
lexware vouchers download <file_id> --format pdf --output voucher.pdf --json
lexware vouchers download <file_id> --format xml --output voucher.xml --json
```

Supported formats are `auto`, `pdf`, and `xml`. For voucher files, `auto` retrieves the original/default representation; use the file ID from the voucher's `files` array. Files are created with owner-only permissions. Existing destinations cause an error unless `--force` is supplied; treat `--force` as an explicit overwrite action.

### Voucher files

```bash
lexware vouchers upload receipt.pdf --dry-run --json
lexware vouchers upload receipt.pdf --json
lexware vouchers attach <voucher_id> supporting-document.pdf --dry-run --json
lexware vouchers attach <voucher_id> supporting-document.pdf --json
```

Supported files are PDF, JPG, JPEG, PNG, and XML, with a maximum size of 5 MiB. `upload` creates a new voucher file; `attach` adds a file to an existing voucher.

### Authentication

Data commands use `LEXWARE_API_KEY` if the environment already provides it and otherwise use the API key stored in the system keychain. Run the requested command without an authentication preflight.

If a data command exits with code 3, authentication or permission is missing. Report the task as blocked and tell the user to run this in their own terminal:

```bash
lexware auth set
```

The command reads the key interactively without displaying it. Never run `lexware auth set` unattended, never pass `--token`, and never request the key through chat. `lexware auth logout` removes the stored key and must only be run when the user explicitly asks to log out.

## Exit Codes

| Code | Meaning |
|------|---------|
| 0 | Success |
| 1 | General error |
| 2 | Resource not found |
| 3 | Authentication or permission error |
| 4 | Input, validation, or conflict error |
| 5 | Rate limit |
| 6 | Network or server error |
