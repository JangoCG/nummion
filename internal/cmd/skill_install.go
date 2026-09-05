package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/JangoCG/nummion/internal/harness"
	"github.com/JangoCG/nummion/skills"
)

const (
	skillFilename        = "SKILL.md"
	installedVersionFile = ".installed-version"
	ownershipMarkerFile  = harness.SkillOwnershipMarker
)

type unmanagedSkillDirError struct{ dir string }

func (e *unmanagedSkillDirError) Error() string {
	return fmt.Sprintf("%s existiert, wurde aber nicht von Nummion angelegt; verschiebe den Ordner, bevor der Nummion-Skill dort installiert wird", e.dir)
}

// claimSkillDir is the ownership gate for every skill write. It claims a
// missing or empty directory, accepts a directory carrying our marker, and
// refuses symlinks, files, and populated directories owned by somebody else.
func claimSkillDir(dir string) error {
	info, err := os.Lstat(dir)
	switch {
	case os.IsNotExist(err):
		if mkErr := os.MkdirAll(dir, 0o755); mkErr != nil { // #nosec G301 -- Public documentation directory, never credentials.
			return fmt.Errorf("Skill-Ordner konnte nicht angelegt werden: %w", mkErr)
		}
	case err != nil:
		return fmt.Errorf("Skill-Ordner konnte nicht geprüft werden: %w", err)
	case info.Mode()&os.ModeSymlink != 0:
		return &unmanagedSkillDirError{dir: dir}
	case !info.IsDir():
		return &unmanagedSkillDirError{dir: dir}
	case !ownedSkillDir(dir):
		entries, readErr := os.ReadDir(dir)
		if readErr != nil {
			return fmt.Errorf("Skill-Ordner konnte nicht geprüft werden: %w", readErr)
		}
		if len(entries) > 0 {
			return &unmanagedSkillDirError{dir: dir}
		}
	}
	return writeOwnershipMarker(dir)
}

func writeOwnershipMarker(dir string) error {
	content := []byte("This skill is managed by Nummion. Manual edits will be overwritten on upgrade.\n")
	if err := writeSkillFile(filepath.Join(dir, ownershipMarkerFile), content); err != nil {
		return fmt.Errorf("Besitzmarker konnte nicht geschrieben werden: %w", err)
	}
	return nil
}

// writeSkillFile refuses a symlink or non-regular final path component so a
// planted link can never redirect an installation outside a claimed folder.
func writeSkillFile(path string, data []byte) error {
	if info, err := os.Lstat(path); err == nil {
		if !info.Mode().IsRegular() {
			return &unmanagedSkillDirError{dir: path}
		}
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("%s konnte nicht geprüft werden: %w", path, err)
	}
	// #nosec G306 G703 -- Public embedded skill data; callers derive paths from operator-owned agent directories and check ownership.
	return os.WriteFile(path, data, 0o644)
}

func ownedSkillDir(dir string) bool {
	return harness.SkillDirOwned(dir)
}

func newSkillInstallCommand(opts *options) *cobra.Command {
	return &cobra.Command{
		Use:   "install",
		Short: "Nummion-Skill global für erkannte Coding-Agents installieren",
		Long:  "Kopiert SKILL.md nach ~/.agents/skills/nummion, verlinkt sie für Claude Code und kopiert sie in den aktiven Codex-Skill-Ordner, sofern die Agents erkannt werden.",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runSkillInstall(cmd, opts)
		},
	}
}

func runSkillInstall(cmd *cobra.Command, opts *options) error {
	skillPath, err := installSkillFiles(opts.version)
	if err != nil {
		return err
	}

	result := map[string]any{"ok": true, "skill_path": skillPath}
	lines := []string{"Nummion-Skill installiert: " + skillPath}

	if harness.DetectClaude() {
		notice, linkErr := linkSkillToClaude()
		if linkErr != nil {
			return linkErr
		}
		home, _ := os.UserHomeDir()
		claudePath := filepath.Join(home, ".claude", "skills", "nummion")
		result["claude_skill_path"] = claudePath
		if notice != "" {
			result["claude_notice"] = notice
			lines = append(lines, "Nummion-Skill für Claude Code kopiert: "+claudePath+" ("+notice+")")
		} else {
			lines = append(lines, "Claude-Code-Symlink angelegt: "+claudePath+" → "+claudeSkillLinkTarget)
		}
	}

	if harness.DetectCodex() {
		codexPath, codexErr := installSkillToCodex()
		if codexErr != nil {
			return codexErr
		}
		result["codex_skill_path"] = codexPath
		lines = append(lines, "Nummion-Skill für Codex kopiert: "+codexPath)
	}

	if notices := retireLegacySkills(); len(notices) > 0 {
		result["legacy_notices"] = notices
		lines = append(lines, notices...)
	}

	printer := opts.printer(cmd)
	if opts.json {
		return printer.Object(result)
	}
	if opts.quiet {
		_, err := fmt.Fprintln(cmd.OutOrStdout(), skillPath)
		return err
	}
	for _, line := range lines {
		if _, err := fmt.Fprintln(cmd.OutOrStdout(), line); err != nil {
			return err
		}
	}
	return nil
}

// installSkillFiles writes the embedded baseline copy and version stamp.
func installSkillFiles(version string) (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("Home-Ordner konnte nicht ermittelt werden: %w", err)
	}
	skillDir := filepath.Join(filepath.Clean(home), ".agents", "skills", "nummion")
	skillPath := filepath.Join(skillDir, skillFilename)

	data, err := skills.FS.ReadFile("nummion/SKILL.md")
	if err != nil {
		return "", fmt.Errorf("eingebetteter Skill konnte nicht gelesen werden: %w", err)
	}
	if err := claimSkillDir(skillDir); err != nil {
		return "", err
	}
	if err := writeSkillFile(skillPath, data); err != nil {
		return "", fmt.Errorf("Skill-Datei konnte nicht geschrieben werden: %w", err)
	}
	if err := writeSkillFile(filepath.Join(skillDir, installedVersionFile), []byte(version)); err != nil {
		return "", fmt.Errorf("Skill-Version konnte nicht geschrieben werden: %w", err)
	}
	return skillPath, nil
}

// claudeSkillLinkTarget is both the relative link target and its provenance:
// only a link with this exact target is considered safe to replace.
var claudeSkillLinkTarget = filepath.Join("..", "..", ".agents", "skills", "nummion")

var makeSkillSymlink = os.Symlink

func linkSkillToClaude() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("Home-Ordner konnte nicht ermittelt werden: %w", err)
	}
	skillDir := filepath.Join(filepath.Clean(home), ".agents", "skills", "nummion")
	linkDir := filepath.Join(filepath.Clean(home), ".claude", "skills")
	linkPath := filepath.Join(linkDir, "nummion")

	if !baselineSkillInstalled() {
		return "", &unmanagedSkillDirError{dir: skillDir}
	}
	if err := os.MkdirAll(linkDir, 0o755); err != nil { // #nosec G301 -- Links to public agent documentation.
		return "", fmt.Errorf("Claude-Skill-Ordner konnte nicht angelegt werden: %w", err)
	}
	if err := removeExistingSkillLink(linkPath); err != nil {
		return "", err
	}
	if err := makeSkillSymlink(claudeSkillLinkTarget, linkPath); err != nil {
		notice := fmt.Sprintf("Symlink nicht verfügbar: %v; Dateien wurden stattdessen kopiert", err)
		if copyErr := copySkillFiles(skillDir, linkPath); copyErr != nil {
			return "", fmt.Errorf("Symlink fehlgeschlagen: %w; Kopie ebenfalls fehlgeschlagen: %v", err, copyErr)
		}
		return notice, nil
	}
	return "", nil
}

func removeExistingSkillLink(path string) error {
	info, err := os.Lstat(path)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("vorhandener Skill-Pfad konnte nicht geprüft werden: %w", err)
	}
	switch {
	case info.Mode()&os.ModeSymlink != 0:
		target, readErr := os.Readlink(path)
		if readErr != nil || target != claudeSkillLinkTarget {
			return &unmanagedSkillDirError{dir: path}
		}
		if err := os.Remove(path); err != nil {
			return fmt.Errorf("vorhandener Skill-Symlink konnte nicht entfernt werden: %w", err)
		}
	case info.IsDir():
		if !isManagedSkillCopy(path) {
			return &unmanagedSkillDirError{dir: path}
		}
		if err := os.RemoveAll(path); err != nil {
			return fmt.Errorf("vorhandene Skill-Kopie konnte nicht entfernt werden: %w", err)
		}
	default:
		return &unmanagedSkillDirError{dir: path}
	}
	return nil
}

func isManagedSkillCopy(path string) bool {
	info, err := os.Lstat(path)
	if err != nil || !info.IsDir() {
		return false
	}
	entries, err := os.ReadDir(path)
	if err != nil {
		return false
	}
	sawMarker := false
	for _, entry := range entries {
		if !entry.Type().IsRegular() {
			return false
		}
		switch entry.Name() {
		case ownershipMarkerFile:
			sawMarker = true
		case skillFilename, installedVersionFile:
		default:
			return false
		}
	}
	return sawMarker
}

func installSkillToCodex() (string, error) {
	skillPath := harness.CodexSkillPath()
	if skillPath == "" {
		return "", fmt.Errorf("Codex-Home-Ordner konnte nicht ermittelt werden")
	}
	data, err := skills.FS.ReadFile("nummion/SKILL.md")
	if err != nil {
		return "", fmt.Errorf("eingebetteter Skill konnte nicht gelesen werden: %w", err)
	}
	if err := claimSkillDir(filepath.Dir(skillPath)); err != nil {
		return "", err
	}
	if err := writeSkillFile(skillPath, data); err != nil {
		return "", fmt.Errorf("Codex-Skill konnte nicht geschrieben werden: %w", err)
	}
	return skillPath, nil
}

func baselineSkillInstalled() bool {
	home, err := os.UserHomeDir()
	if err != nil {
		return false
	}
	skillDir := filepath.Join(filepath.Clean(home), ".agents", "skills", "nummion")
	return harness.RegularSkillFile(filepath.Join(skillDir, skillFilename)) && ownedSkillDir(skillDir)
}

func installedSkillVersion() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	data, err := os.ReadFile(filepath.Join(filepath.Clean(home), ".agents", "skills", "nummion", installedVersionFile))
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(data))
}

func copySkillFiles(src, dst string) error {
	if err := claimSkillDir(dst); err != nil {
		return err
	}
	for _, name := range []string{skillFilename, installedVersionFile, ownershipMarkerFile} {
		data, err := os.ReadFile(filepath.Join(src, name)) // #nosec G304 -- Fixed skill filenames under an operator-owned source directory.
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return err
		}
		if err := writeSkillFile(filepath.Join(dst, name), data); err != nil {
			return err
		}
	}
	return nil
}
