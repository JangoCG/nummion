package cmd

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/spf13/cobra"

	"lexware-cli/internal/payload"
)

func newInvoicesCommand(opts *options) *cobra.Command {
	invoices := &cobra.Command{Use: "invoices", Short: "Rechnungen verwalten", Args: cobra.NoArgs}
	invoices.AddCommand(newVoucherListCommand(opts, "invoice"))
	invoices.AddCommand(newInvoicesGetCommand(opts))
	invoices.AddCommand(newInvoicesCreateCommand(opts))
	invoices.AddCommand(newInvoicesDownloadCommand(opts))
	return invoices
}

func newInvoicesGetCommand(opts *options) *cobra.Command {
	return &cobra.Command{
		Use:   "get ID",
		Short: "Eine Rechnung abrufen",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := opts.client()
			if err != nil {
				return err
			}
			raw, err := client.JSON(cmd.Context(), http.MethodGet, "/v1/invoices/"+url.PathEscape(args[0]), nil, nil)
			if err != nil {
				return err
			}
			return opts.printer(cmd).Data(raw)
		},
	}
}

func newInvoicesCreateCommand(opts *options) *cobra.Command {
	var from string
	var finalize bool
	var dryRun bool
	command := &cobra.Command{
		Use:   "create --from DATEI",
		Short: "Rechnung aus JSON erstellen; standardmäßig als Entwurf",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			body, object, err := payload.ReadJSON(from, cmd.InOrStdin())
			if err != nil {
				return err
			}
			query := url.Values{}
			if finalize {
				query.Set("finalize", "true")
			}
			if dryRun {
				return opts.printer(cmd).Object(map[string]any{
					"dryRun":   true,
					"method":   "POST",
					"path":     "/v1/invoices",
					"finalize": finalize,
					"body":     object,
				})
			}
			client, err := opts.client()
			if err != nil {
				return err
			}
			raw, err := client.JSONBytes(cmd.Context(), http.MethodPost, "/v1/invoices", query, body)
			if err != nil {
				return err
			}
			return opts.printer(cmd).Data(raw)
		},
	}
	command.Flags().StringVarP(&from, "from", "f", "", "JSON-Datei oder '-' für stdin")
	command.Flags().BoolVar(&finalize, "finalize", false, "Rechnung sofort finalisieren statt als Entwurf anzulegen")
	command.Flags().BoolVar(&dryRun, "dry-run", false, "Anfrage anzeigen, aber nicht senden")
	_ = command.MarkFlagRequired("from")
	return command
}

func newInvoicesDownloadCommand(opts *options) *cobra.Command {
	var outputPath string
	var format string
	var force bool
	command := &cobra.Command{
		Use:   "download ID",
		Short: "Rechnungsdatei als PDF oder XML herunterladen",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			accept := "*/*"
			extension := ".pdf"
			switch strings.ToLower(format) {
			case "auto":
			case "pdf":
				accept = "application/pdf"
			case "xml":
				accept = "application/xml"
				extension = ".xml"
			default:
				return fmt.Errorf("ungültiges Format %q; erlaubt sind auto, pdf und xml", format)
			}
			client, err := opts.client()
			if err != nil {
				return err
			}
			req, err := client.NewRequest(cmd.Context(), http.MethodGet, "/v1/invoices/"+url.PathEscape(args[0])+"/file", nil, nil)
			if err != nil {
				return err
			}
			req.Header.Set("Accept", accept)
			resp, err := client.Do(req)
			if err != nil {
				return err
			}
			target, err := writeDownload(resp, outputPath, "invoice-"+safeBaseName(args[0])+extension, force)
			if err != nil {
				return err
			}
			if opts.json {
				return opts.printer(cmd).Object(map[string]any{"ok": true, "path": target})
			}
			if opts.quiet {
				_, err = fmt.Fprintln(cmd.OutOrStdout(), target)
				return err
			}
			return opts.printer(cmd).Success("Gespeichert: " + target)
		},
	}
	command.Flags().StringVarP(&outputPath, "output", "o", "", "Zieldatei; standardmäßig der Server-Dateiname")
	command.Flags().StringVar(&format, "format", "auto", "Dateiformat: auto, pdf oder xml")
	command.Flags().BoolVar(&force, "force", false, "vorhandene Zieldatei überschreiben")
	return command
}
