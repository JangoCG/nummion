package cmd

import (
	"fmt"
	"net/url"
	"strconv"

	"github.com/spf13/cobra"
)

func newVoucherListCommand(opts *options, defaultType string) *cobra.Command {
	var (
		voucherType     string
		voucherStatus   string
		voucherNumber   string
		contactID       string
		voucherDateFrom string
		voucherDateTo   string
		createdDateFrom string
		createdDateTo   string
		updatedDateFrom string
		updatedDateTo   string
		archived        bool
		year            int
		page            int
		size            int
		all             bool
	)
	var command *cobra.Command
	command = &cobra.Command{
		Use:   "list",
		Short: "Belege über die Voucherlist auflisten und filtern",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if err := validatePaging(page, size); err != nil {
				return err
			}
			if command.Flags().Changed("year") {
				if year < 1000 || year > 9999 {
					return fmt.Errorf("--year muss ein vierstelliges Jahr sein")
				}
				if command.Flags().Changed("voucher-date-from") || command.Flags().Changed("voucher-date-to") {
					return fmt.Errorf("--year kann nicht mit --voucher-date-from oder --voucher-date-to kombiniert werden")
				}
				voucherDateFrom = fmt.Sprintf("%04d-01-01", year)
				voucherDateTo = fmt.Sprintf("%04d-12-31", year)
				all = true
				page = 0
			}
			client, err := opts.client()
			if err != nil {
				return err
			}
			query := url.Values{
				"voucherType":   {voucherType},
				"voucherStatus": {voucherStatus},
				"page":          {strconv.Itoa(page)},
				"size":          {strconv.Itoa(size)},
			}
			optional := map[string]string{
				"voucherNumber":   voucherNumber,
				"contactId":       contactID,
				"voucherDateFrom": voucherDateFrom,
				"voucherDateTo":   voucherDateTo,
				"createdDateFrom": createdDateFrom,
				"createdDateTo":   createdDateTo,
				"updatedDateFrom": updatedDateFrom,
				"updatedDateTo":   updatedDateTo,
			}
			for key, value := range optional {
				if value != "" {
					query.Set(key, value)
				}
			}
			if command.Flags().Changed("archived") {
				query.Set("archived", strconv.FormatBool(archived))
			}
			if all && size < 250 {
				query.Set("size", "250")
			}
			raw, result, err := fetchPage(cmd.Context(), client, "/v1/voucherlist", query, all)
			if err != nil {
				return err
			}
			printer := opts.printer(cmd)
			if opts.json || opts.quiet {
				return printer.Data(raw)
			}
			rows := make([][]string, 0, len(result.Content))
			for _, voucher := range result.Content {
				rows = append(rows, []string{
					stringValue(voucher["id"]),
					stringValue(voucher["voucherType"]),
					stringValue(voucher["voucherStatus"]),
					stringValue(voucher["voucherNumber"]),
					stringValue(voucher["contactName"]),
					stringValue(voucher["totalAmount"]),
					stringValue(voucher["openAmount"]),
					stringValue(voucher["voucherDate"]),
				})
			}
			if err := printer.Table([]string{"ID", "TYP", "STATUS", "NUMMER", "KONTAKT", "SUMME", "OFFEN", "DATUM"}, rows); err != nil {
				return err
			}
			_, err = fmt.Fprintf(cmd.OutOrStdout(), "\n%d von %d Belegen\n", len(result.Content), result.TotalElements)
			return err
		},
	}
	command.Flags().StringVar(&voucherType, "type", defaultType, "Belegtyp oder kommaseparierte Typen")
	command.Flags().StringVar(&voucherStatus, "status", "any", "Status oder kommaseparierte Statuswerte")
	command.Flags().StringVar(&voucherNumber, "number", "", "Belegnummer")
	command.Flags().StringVar(&contactID, "contact-id", "", "Kontakt-ID")
	command.Flags().StringVar(&voucherDateFrom, "voucher-date-from", "", "Belegdatum ab (YYYY-MM-DD)")
	command.Flags().StringVar(&voucherDateTo, "voucher-date-to", "", "Belegdatum bis (YYYY-MM-DD)")
	command.Flags().IntVarP(&year, "year", "y", 0, "alle Belege eines Kalenderjahres abrufen, z. B. 2025")
	command.Flags().StringVar(&createdDateFrom, "created-from", "", "Erstellungsdatum ab (YYYY-MM-DD)")
	command.Flags().StringVar(&createdDateTo, "created-to", "", "Erstellungsdatum bis (YYYY-MM-DD)")
	command.Flags().StringVar(&updatedDateFrom, "updated-from", "", "Änderungsdatum ab (YYYY-MM-DD)")
	command.Flags().StringVar(&updatedDateTo, "updated-to", "", "Änderungsdatum bis (YYYY-MM-DD)")
	command.Flags().BoolVar(&archived, "archived", false, "archivierte bzw. nicht archivierte Belege filtern")
	command.Flags().IntVar(&page, "page", 0, "Seitennummer (ab 0)")
	command.Flags().IntVar(&size, "size", 25, "Seitengröße (maximal 250)")
	command.Flags().BoolVar(&all, "all", false, "alle Seiten abrufen; bei --year automatisch aktiv")
	return command
}
