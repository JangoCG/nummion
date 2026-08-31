package cmd

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"golang.org/x/term"

	"lexware-cli/internal/api"
	"lexware-cli/internal/credentials"
)

func newAuthCommand(opts *options) *cobra.Command {
	auth := &cobra.Command{Use: "auth", Short: "API-Key verwalten", Args: cobra.NoArgs}
	auth.AddCommand(newAuthSetCommand(opts))
	auth.AddCommand(newAuthStatusCommand(opts))
	auth.AddCommand(newAuthLogoutCommand(opts))
	return auth
}

func newAuthSetCommand(opts *options) *cobra.Command {
	var tokenFlag string
	command := &cobra.Command{
		Use:   "set",
		Short: "API-Key sicher im System-Schlüsselbund speichern",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			token := strings.TrimSpace(tokenFlag)
			if token == "" {
				var err error
				token, err = readSecret(cmd)
				if err != nil {
					return err
				}
			}
			if err := credentials.Set(token); err != nil {
				return err
			}
			return opts.printer(cmd).Success("API-Key wurde im System-Schlüsselbund gespeichert.")
		},
	}
	command.Flags().StringVar(&tokenFlag, "token", "", "API-Key direkt übergeben (unsicher: kann in der Shell-History landen)")
	return command
}

func readSecret(cmd *cobra.Command) (string, error) {
	if file, ok := cmd.InOrStdin().(*os.File); ok && term.IsTerminal(int(file.Fd())) {
		fmt.Fprint(cmd.ErrOrStderr(), "Lexware API-Key: ")
		secret, err := term.ReadPassword(int(file.Fd()))
		fmt.Fprintln(cmd.ErrOrStderr())
		if err != nil {
			return "", fmt.Errorf("API-Key konnte nicht gelesen werden: %w", err)
		}
		return strings.TrimSpace(string(secret)), nil
	}
	reader := bufio.NewReader(io.LimitReader(cmd.InOrStdin(), 64<<10))
	secret, err := io.ReadAll(reader)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(secret)), nil
}

func newAuthStatusCommand(opts *options) *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "API-Key prüfen und verbundenes Konto anzeigen",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			resolved, err := credentials.Resolve()
			if err != nil {
				return err
			}
			client, err := api.New(opts.baseURL, resolved.Token, "lexware-cli/"+opts.version, opts.timeout)
			if err != nil {
				return err
			}
			raw, err := client.JSON(cmd.Context(), "GET", "/v1/profile", nil, nil)
			if err != nil {
				return err
			}
			if opts.json || opts.quiet {
				return opts.printer(cmd).Data(raw)
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Authentifiziert über %s.\n", resolved.Source)
			return opts.printer(cmd).Data(raw)
		},
	}
}

func newAuthLogoutCommand(opts *options) *cobra.Command {
	return &cobra.Command{
		Use:   "logout",
		Short: "Gespeicherten API-Key entfernen",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if err := credentials.Delete(); err != nil && !errors.Is(err, credentials.ErrNotConfigured) {
				return err
			}
			return opts.printer(cmd).Success("Gespeicherter API-Key wurde entfernt.")
		},
	}
}
