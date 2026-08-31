# lexware-cli

Eine inoffizielle, lokal laufende Kommandozeile für die Lexware Office Public API.

Das Projekt ist für den privaten Gebrauch gedacht und weder mit Lexware noch mit der Haufe Group verbunden. `Lexware` und `Lexware Office` sind Marken ihrer jeweiligen Inhaber.

## Funktionen

- API-Key sicher im macOS-Systemschlüsselbund speichern
- Profil und verfügbare Business-Funktionen abrufen
- Kontakte auflisten, filtern, erstellen und aktualisieren
- Rechnungen auflisten, abrufen, als Entwurf oder finalisiert erstellen und herunterladen
- Buchhaltungsbelege auflisten, abrufen, hochladen und Dateien anhängen
- Tabellen für Menschen und kompaktes JSON für Skripte
- lokales Rate-Limiting sowie Retry bei HTTP 429
- `--dry-run` für schreibende Arbeitsabläufe
- sichere Kontakt-Updates durch Zusammenführen mit dem aktuellen Datensatz und optimistisches Locking über das Lexware-`version`-Feld

## Voraussetzungen und Build

Das Repository enthält eine `mise`-Konfiguration für Go:

```bash
cd /Users/cengiz/Developer/lexware-cli
mise install
mise exec -- make check
```

Das Binary liegt danach unter `bin/lexware`.

Optional lokal installieren:

```bash
install -m 0755 bin/lexware ~/.local/bin/lexware
```

## API-Key einrichten

Einen privaten API-Key erzeugst du in Lexware unter:

<https://app.lexware.de/addons/public-api>

Anschließend wird er ohne Anzeige im macOS-Schlüsselbund gespeichert:

```bash
./bin/lexware auth set
./bin/lexware auth status
```

Für CI oder kurzlebige Shells kann der Key stattdessen über `LEXWARE_API_KEY` gesetzt werden. Die Umgebungsvariable hat Vorrang vor dem Schlüsselbund.

## Beispiele

```bash
# Konto und Kontakte
lexware profile
lexware contacts list --name Muster
lexware contacts get CONTACT_ID --json
lexware contacts create --from examples/contact.json --dry-run
lexware contacts create --from examples/contact.json

# Update führt die Änderungen standardmäßig mit dem aktuellen Kontakt zusammen
# und verwendet dessen version-Feld.
lexware contacts update CONTACT_ID --from contact-update.json --dry-run
lexware contacts update CONTACT_ID --from contact-update.json

# Nur wenn bewusst ein vollständiger Ersatz gesendet werden soll:
lexware contacts update CONTACT_ID --from complete-contact.json --replace

# Rechnungen: Ohne --finalize entsteht ein Entwurf.
lexware invoices list --status any
lexware invoices create --from examples/invoice.json --dry-run
lexware invoices create --from examples/invoice.json
lexware invoices create --from examples/invoice.json --finalize
lexware invoices download INVOICE_ID --format pdf

# Eingangsbelege
lexware vouchers list --type purchaseinvoice --status any
lexware vouchers upload ./eingangsrechnung.pdf --dry-run
lexware vouchers upload ./eingangsrechnung.pdf
lexware vouchers attach VOUCHER_ID ./anlage.pdf
```

JSON kann auch über stdin übergeben werden:

```bash
generate-contact | lexware contacts create --from - --json
```

## Sicherheitsregeln

- `auth set` fragt den Key verdeckt ab. `--token` ist nur für Sonderfälle gedacht, weil Shell-History und Prozesslisten den Wert offenlegen können.
- Downloads werden mit Dateirechten `0600` angelegt und ohne `--force` niemals überschrieben.
- Beleg-Uploads werden vorab auf erlaubtes Format und die Lexware-Grenze von 5 MiB geprüft.
- `invoices create` erzeugt standardmäßig einen Entwurf. Eine sofortige Finalisierung erfordert ausdrücklich `--finalize`.
- Bei HTTP 504 werden schreibende Requests nicht automatisch wiederholt, da Lexware darauf hinweist, dass die Verarbeitung bereits erfolgt sein kann.

## Exit-Codes

| Code | Bedeutung |
| ---: | --- |
| 0 | Erfolg |
| 1 | allgemeiner Fehler |
| 2 | Ressource nicht gefunden |
| 3 | Authentifizierung oder Berechtigung |
| 4 | Eingabe-, Validierungs- oder Konfliktfehler |
| 5 | Rate-Limit |
| 6 | Netzwerk- oder Serverfehler |

## Hinweise zur API

Die Implementierung verwendet ausschließlich die dokumentierte Basis-URL `https://api.lexware.io`. Die API-Antworten bleiben bei `--json` unverändert, damit Skripte nicht von zusätzlichen CLI-Hüllen abhängig werden.

- Dokumentation: <https://developers.lexware.io/docs/>
- API-Nutzungsbedingungen: <https://agb.lexware.de/lexware-office/public-api-lizenz--und-nutzungsbedingungen>

## Lizenz

MIT. Siehe [LICENSE](LICENSE).
