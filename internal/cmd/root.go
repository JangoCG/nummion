package cmd

import (
	"context"
	"errors"
	"fmt"
	"net"
	"time"

	"github.com/spf13/cobra"

	"lexware-cli/internal/api"
	"lexware-cli/internal/credentials"
	"lexware-cli/internal/output"
)

type options struct {
	baseURL string
	timeout time.Duration
	json    bool
	quiet   bool
	version string
}

func NewRootCommand(version string) *cobra.Command {
	opts := &options{version: version}
	root := &cobra.Command{
		Use:           "lexware",
		Short:         "Inoffizielle CLI für die Lexware Office Public API",
		SilenceUsage:  true,
		SilenceErrors: true,
		Args:          cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return cmd.Help()
		},
	}
	root.SetVersionTemplate("lexware {{.Version}}\n")
	root.Version = version
	root.PersistentFlags().StringVar(&opts.baseURL, "base-url", api.DefaultBaseURL, "API-Basis-URL")
	root.PersistentFlags().DurationVar(&opts.timeout, "timeout", 30*time.Second, "HTTP-Timeout")
	root.PersistentFlags().BoolVar(&opts.json, "json", false, "API-Antwort als kompaktes JSON ausgeben")
	root.PersistentFlags().BoolVarP(&opts.quiet, "quiet", "q", false, "nur maschinenlesbare Ergebnisse ausgeben")

	root.AddCommand(newAuthCommand(opts))
	root.AddCommand(newProfileCommand(opts))
	root.AddCommand(newContactsCommand(opts))
	root.AddCommand(newInvoicesCommand(opts))
	root.AddCommand(newVouchersCommand(opts))
	return root
}

func (o *options) client() (*api.Client, error) {
	resolved, err := credentials.Resolve()
	if err != nil {
		return nil, err
	}
	return api.New(o.baseURL, resolved.Token, "lexware-cli/"+o.version, o.timeout)
}

func (o *options) printer(cmd *cobra.Command) output.Printer {
	return output.Printer{Out: cmd.OutOrStdout(), JSON: o.json, Quiet: o.quiet}
}

func ExitCode(err error) int {
	if err == nil {
		return 0
	}
	if errors.Is(err, credentials.ErrNotConfigured) {
		return 3
	}
	if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
		return 6
	}
	var apiErr *api.Error
	if errors.As(err, &apiErr) {
		switch apiErr.StatusCode {
		case 401, 403:
			return 3
		case 404:
			return 2
		case 400, 406, 409, 415:
			return 4
		case 429:
			return 5
		default:
			if apiErr.StatusCode >= 500 {
				return 6
			}
		}
	}
	var netErr net.Error
	if errors.As(err, &netErr) {
		return 6
	}
	return 1
}

func requireID(value string) error {
	if value == "" {
		return fmt.Errorf("ID darf nicht leer sein")
	}
	return nil
}
