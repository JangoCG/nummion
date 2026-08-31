package cmd

import (
	"bytes"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"lexware-cli/internal/api"
)

const maxVoucherFileSize = 5 << 20

func newVouchersCommand(opts *options) *cobra.Command {
	vouchers := &cobra.Command{Use: "vouchers", Short: "Buchhaltungsbelege verwalten", Args: cobra.NoArgs}
	vouchers.AddCommand(newVoucherListCommand(opts, "any"))
	vouchers.AddCommand(newVouchersGetCommand(opts))
	vouchers.AddCommand(newVouchersUploadCommand(opts))
	vouchers.AddCommand(newVouchersAttachCommand(opts))
	return vouchers
}

func newVouchersGetCommand(opts *options) *cobra.Command {
	return &cobra.Command{
		Use:   "get ID",
		Short: "Einen Buchhaltungsbeleg abrufen",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := opts.client()
			if err != nil {
				return err
			}
			raw, err := client.JSON(cmd.Context(), http.MethodGet, "/v1/vouchers/"+url.PathEscape(args[0]), nil, nil)
			if err != nil {
				return err
			}
			return opts.printer(cmd).Data(raw)
		},
	}
}

func newVouchersUploadCommand(opts *options) *cobra.Command {
	var dryRun bool
	command := &cobra.Command{
		Use:   "upload DATEI",
		Short: "PDF, JPG, PNG oder XML als neuen Beleg hochladen",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			info, err := validateVoucherFile(args[0])
			if err != nil {
				return err
			}
			if dryRun {
				return opts.printer(cmd).Object(map[string]any{
					"dryRun": true,
					"method": "POST",
					"path":   "/v1/files",
					"file":   args[0],
					"bytes":  info.Size(),
					"type":   "voucher",
				})
			}
			return uploadVoucherFile(cmd, opts, "/v1/files", args[0], true)
		},
	}
	command.Flags().BoolVar(&dryRun, "dry-run", false, "Datei prüfen, aber nicht hochladen")
	return command
}

func newVouchersAttachCommand(opts *options) *cobra.Command {
	var dryRun bool
	command := &cobra.Command{
		Use:   "attach ID DATEI",
		Short: "Datei an einen bestehenden Buchhaltungsbeleg anhängen",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			info, err := validateVoucherFile(args[1])
			if err != nil {
				return err
			}
			path := "/v1/vouchers/" + url.PathEscape(args[0]) + "/files"
			if dryRun {
				return opts.printer(cmd).Object(map[string]any{
					"dryRun": true,
					"method": "POST",
					"path":   path,
					"file":   args[1],
					"bytes":  info.Size(),
				})
			}
			return uploadVoucherFile(cmd, opts, path, args[1], false)
		},
	}
	command.Flags().BoolVar(&dryRun, "dry-run", false, "Datei prüfen, aber nicht hochladen")
	return command
}

func validateVoucherFile(path string) (os.FileInfo, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("Belegdatei konnte nicht gelesen werden: %w", err)
	}
	if !info.Mode().IsRegular() {
		return nil, fmt.Errorf("%q ist keine reguläre Datei", path)
	}
	if info.Size() == 0 {
		return nil, fmt.Errorf("Belegdatei ist leer")
	}
	if info.Size() > maxVoucherFileSize {
		return nil, fmt.Errorf("Belegdatei ist %d Bytes groß; Lexware erlaubt maximal 5 MiB", info.Size())
	}
	switch strings.ToLower(filepath.Ext(path)) {
	case ".pdf", ".jpg", ".jpeg", ".png", ".xml":
	default:
		return nil, fmt.Errorf("nicht unterstütztes Dateiformat %q; erlaubt sind PDF, JPG, PNG und XML", filepath.Ext(path))
	}
	return info, nil
}

func uploadVoucherFile(cmd *cobra.Command, opts *options, path, filePath string, includeType bool) error {
	file, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	part, err := writer.CreateFormFile("file", filepath.Base(filePath))
	if err != nil {
		return err
	}
	if _, err := io.Copy(part, file); err != nil {
		return err
	}
	if includeType {
		if err := writer.WriteField("type", "voucher"); err != nil {
			return err
		}
	}
	if err := writer.Close(); err != nil {
		return err
	}

	client, err := opts.client()
	if err != nil {
		return err
	}
	req, err := client.NewRequest(cmd.Context(), http.MethodPost, path, nil, &body)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	raw, err := api.ReadJSONResponse(resp)
	if err != nil {
		return err
	}
	return opts.printer(cmd).Data(raw)
}
