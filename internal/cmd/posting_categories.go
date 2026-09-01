package cmd

import (
	"net/http"

	"github.com/spf13/cobra"
)

func newPostingCategoriesCommand(opts *options) *cobra.Command {
	postingCategories := &cobra.Command{
		Use:   "posting-categories",
		Short: "Buchungskategorien verwalten",
		Args:  cobra.NoArgs,
	}
	postingCategories.AddCommand(newPostingCategoriesListCommand(opts))
	return postingCategories
}

func newPostingCategoriesListCommand(opts *options) *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "Verfügbare Buchungskategorien auflisten",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			client, err := opts.client()
			if err != nil {
				return err
			}
			raw, err := client.JSON(cmd.Context(), http.MethodGet, "/v1/posting-categories", nil, nil)
			if err != nil {
				return err
			}
			return opts.printer(cmd).Data(raw)
		},
	}
}
