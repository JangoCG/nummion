package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/JangoCG/nummion/skills"
)

func newSkillCommand(opts *options) *cobra.Command {
	command := &cobra.Command{
		Use:   "skill",
		Short: "Eingebetteten Agent-Skill anzeigen oder installieren",
		Long:  "Gibt die in Nummion eingebettete SKILL.md aus oder installiert sie für erkannte Coding-Agents.",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			data, err := skills.FS.ReadFile("nummion/SKILL.md")
			if err != nil {
				return fmt.Errorf("eingebetteter Skill konnte nicht gelesen werden: %w", err)
			}
			if _, err := cmd.OutOrStdout().Write(data); err != nil {
				return fmt.Errorf("Skill konnte nicht ausgegeben werden: %w", err)
			}
			return nil
		},
	}
	command.AddCommand(newSkillInstallCommand(opts))
	return command
}
