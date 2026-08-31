package cmd

import (
	"net/http"

	"github.com/spf13/cobra"
)

func newProfileCommand(opts *options) *cobra.Command {
	return &cobra.Command{
		Use:   "profile",
		Short: "Lexware-Profil und verfügbare Business-Funktionen anzeigen",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			client, err := opts.client()
			if err != nil {
				return err
			}
			raw, err := client.JSON(cmd.Context(), http.MethodGet, "/v1/profile", nil, nil)
			if err != nil {
				return err
			}
			return opts.printer(cmd).Data(raw)
		},
	}
}
