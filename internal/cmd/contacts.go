package cmd

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"

	"github.com/spf13/cobra"

	"github.com/JangoCG/nummion/internal/payload"
)

func newContactsCommand(opts *options) *cobra.Command {
	contacts := &cobra.Command{Use: "contacts", Short: "Kontakte verwalten", Args: cobra.NoArgs}
	contacts.AddCommand(newContactsListCommand(opts))
	contacts.AddCommand(newContactsGetCommand(opts))
	contacts.AddCommand(newContactsCreateCommand(opts))
	contacts.AddCommand(newContactsUpdateCommand(opts))
	return contacts
}

func newContactsListCommand(opts *options) *cobra.Command {
	var (
		name     string
		email    string
		number   int
		customer bool
		vendor   bool
		page     int
		size     int
		all      bool
	)
	var command *cobra.Command
	command = &cobra.Command{
		Use:   "list",
		Short: "Kontakte auflisten und filtern",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if err := validatePaging(page, size); err != nil {
				return err
			}
			client, err := opts.client()
			if err != nil {
				return err
			}
			query := url.Values{
				"page": {strconv.Itoa(page)},
				"size": {strconv.Itoa(size)},
			}
			if name != "" {
				query.Set("name", name)
			}
			if email != "" {
				query.Set("email", email)
			}
			if command.Flags().Changed("number") {
				query.Set("number", strconv.Itoa(number))
			}
			if command.Flags().Changed("customer") {
				query.Set("customer", strconv.FormatBool(customer))
			}
			if command.Flags().Changed("vendor") {
				query.Set("vendor", strconv.FormatBool(vendor))
			}
			if all && size < 250 {
				query.Set("size", "250")
			}
			raw, result, err := fetchPage(cmd.Context(), client, "/v1/contacts", query, all)
			if err != nil {
				return err
			}
			printer := opts.printer(cmd)
			if opts.json || opts.quiet {
				return printer.Data(raw)
			}
			rows := make([][]string, 0, len(result.Content))
			for _, contact := range result.Content {
				rows = append(rows, []string{
					stringValue(contact["id"]),
					contactName(contact),
					contactNumber(contact),
					stringValue(contact["version"]),
					stringValue(contact["archived"]),
				})
			}
			if err := printer.Table([]string{"ID", "NAME", "NUMMER", "VERSION", "ARCHIVIERT"}, rows); err != nil {
				return err
			}
			_, err = fmt.Fprintf(cmd.OutOrStdout(), "\n%d von %d Kontakten\n", len(result.Content), result.TotalElements)
			return err
		},
	}
	command.Flags().StringVar(&name, "name", "", "Name (mindestens 3 Zeichen)")
	command.Flags().StringVar(&email, "email", "", "E-Mail-Adresse (mindestens 3 Zeichen)")
	command.Flags().IntVar(&number, "number", 0, "Kunden- oder Lieferantennummer")
	command.Flags().BoolVar(&customer, "customer", false, "nach Kundenrolle filtern")
	command.Flags().BoolVar(&vendor, "vendor", false, "nach Lieferantenrolle filtern")
	command.Flags().IntVar(&page, "page", 0, "Seitennummer (ab 0)")
	command.Flags().IntVar(&size, "size", 25, "Seitengröße (maximal 250)")
	command.Flags().BoolVar(&all, "all", false, "alle Seiten abrufen")
	return command
}

func newContactsGetCommand(opts *options) *cobra.Command {
	return &cobra.Command{
		Use:   "get ID",
		Short: "Einen Kontakt abrufen",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := opts.client()
			if err != nil {
				return err
			}
			raw, err := client.JSON(cmd.Context(), http.MethodGet, "/v1/contacts/"+url.PathEscape(args[0]), nil, nil)
			if err != nil {
				return err
			}
			return opts.printer(cmd).Data(raw)
		},
	}
}

func newContactsCreateCommand(opts *options) *cobra.Command {
	var from string
	var dryRun bool
	var command *cobra.Command
	command = &cobra.Command{
		Use:   "create --from DATEI",
		Short: "Kontakt aus einer JSON-Datei erstellen",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			body, object, err := payload.ReadJSON(from, cmd.InOrStdin())
			if err != nil {
				return err
			}
			if _, exists := object["version"]; !exists {
				object["version"] = 0
				body, err = payload.MarshalObject(object)
				if err != nil {
					return err
				}
			}
			if dryRun {
				return opts.printer(cmd).Object(map[string]any{"dryRun": true, "method": "POST", "path": "/v1/contacts", "body": object})
			}
			client, err := opts.client()
			if err != nil {
				return err
			}
			raw, err := client.JSONBytes(cmd.Context(), http.MethodPost, "/v1/contacts", nil, body)
			if err != nil {
				return err
			}
			return opts.printer(cmd).Data(raw)
		},
	}
	command.Flags().StringVarP(&from, "from", "f", "", "JSON-Datei oder '-' für stdin")
	command.Flags().BoolVar(&dryRun, "dry-run", false, "Anfrage anzeigen, aber nicht senden")
	_ = command.MarkFlagRequired("from")
	return command
}

func newContactsUpdateCommand(opts *options) *cobra.Command {
	var from string
	var dryRun bool
	var version int64
	var replace bool
	var command *cobra.Command
	command = &cobra.Command{
		Use:   "update ID --from DATEI",
		Short: "Kontakt aktualisieren; die aktuelle Version wird bei Bedarf automatisch geladen",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := requireID(args[0]); err != nil {
				return err
			}
			_, object, err := payload.ReadJSON(from, cmd.InOrStdin())
			if err != nil {
				return err
			}
			_, payloadHasVersion := object["version"]
			if !replace || (!payloadHasVersion && !command.Flags().Changed("version")) {
				client, err := opts.client()
				if err != nil {
					return err
				}
				currentRaw, err := client.JSON(cmd.Context(), http.MethodGet, "/v1/contacts/"+url.PathEscape(args[0]), nil, nil)
				if err != nil {
					return err
				}
				var current map[string]any
				if err := json.Unmarshal(currentRaw, &current); err != nil {
					return err
				}
				currentVersion, exists := current["version"]
				if !exists {
					return fmt.Errorf("Kontaktantwort enthält kein version-Feld")
				}
				if !replace {
					delete(current, "id")
					delete(current, "organizationId")
					delete(current, "createdDate")
					delete(current, "updatedDate")
					delete(current, "archived")
					delete(current, "version")
					object = mergeObjects(current, object)
				}
				if !replace || !payloadHasVersion {
					object["version"] = currentVersion
				}
			}
			if command.Flags().Changed("version") {
				object["version"] = version
			}
			body, err := payload.MarshalObject(object)
			if err != nil {
				return err
			}
			path := "/v1/contacts/" + url.PathEscape(args[0])
			if dryRun {
				return opts.printer(cmd).Object(map[string]any{"dryRun": true, "method": "PUT", "path": path, "body": object})
			}
			client, err := opts.client()
			if err != nil {
				return err
			}
			raw, err := client.JSONBytes(cmd.Context(), http.MethodPut, path, nil, body)
			if err != nil {
				return err
			}
			return opts.printer(cmd).Data(raw)
		},
	}
	command.Flags().StringVarP(&from, "from", "f", "", "JSON-Datei oder '-' für stdin")
	command.Flags().Int64Var(&version, "version", 0, "bekannte Lexware-Version explizit setzen")
	command.Flags().BoolVar(&replace, "replace", false, "JSON unverändert als vollständigen Kontakt senden statt mit dem aktuellen Kontakt zusammenzuführen")
	command.Flags().BoolVar(&dryRun, "dry-run", false, "Anfrage anzeigen, aber nicht senden")
	_ = command.MarkFlagRequired("from")
	return command
}
